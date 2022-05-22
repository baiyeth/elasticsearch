package es

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/olivere/elastic/v7"
)

var (
	MaxConnsPerHost       = 10
	MaxIdleConnsPerHost   = 10
	ResponseHeaderTimeout = time.Millisecond * 6000
)

type Config struct {
	Addresses  []string    // A list of Elasticsearch nodes to use.
	Username   string      // Username for HTTP Basic Authentication.
	Password   string      // Password for HTTP Basic Authentication.
	Header     http.Header // Global HTTP request header.
	MaxRetries int         // Default: 3.
}

type ElasticSearch struct {
	ctx context.Context
	cli *elastic.Client
}

func WithUrls(addresses ...string) elastic.ClientOptionFunc {
	return elastic.SetURL(addresses...)
}

func WithAuth(username, password string) elastic.ClientOptionFunc {
	return elastic.SetBasicAuth(username, password)
}

func WithMaxRetries(retries int) elastic.ClientOptionFunc {
	return elastic.SetMaxRetries(retries)
}

func WithHeader(h http.Header) elastic.ClientOptionFunc {
	return elastic.SetHeaders(h)
}

func WithGzip(flag bool) elastic.ClientOptionFunc {
	return elastic.SetGzip(flag)
}

func WithSniff(flag bool) elastic.ClientOptionFunc {
	return elastic.SetSniff(flag)
}

func WithHttpClient(timeout time.Duration) elastic.ClientOptionFunc {
	return elastic.SetHttpClient(&http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:       MaxConnsPerHost,
			MaxIdleConnsPerHost:   MaxIdleConnsPerHost,
			ResponseHeaderTimeout: timeout,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS11,
				// ...
			},
			// ...,
		},
	})
}

func NewClient(ctx context.Context, addresses []string, options ...elastic.ClientOptionFunc) *ElasticSearch {
	options = append(options, WithUrls(addresses...))
	cli, err := elastic.NewClient(options...)
	if err != nil {
		return nil
	}
	return &ElasticSearch{
		ctx: ctx,
		cli: cli,
	}
}

func NewDefaultClient(ctx context.Context, username, password string, addresses []string) *ElasticSearch {
	var options []elastic.ClientOptionFunc
	options = append(options, WithUrls(addresses...),
		WithAuth(username, password),
		WithGzip(true),
		WithSniff(false),
		WithHttpClient(time.Millisecond*30000),
		WithMaxRetries(5))
	cli, err := elastic.NewClient(options...)
	if err != nil {
		return nil
	}
	return &ElasticSearch{
		ctx: ctx,
		cli: cli,
	}
}

func (es *ElasticSearch) Ping(url string) (*elastic.PingResult, int, error) {
	return es.cli.Ping(url).Do(es.ctx)
}

// CreateIndices create Indices with body
// {
//     "aliases": {
//         "book":{}
//     },
//     "settings": {
//         "analysis": {
//             "normalizer": {
//                 "lowercase": {
//                     "type": "custom",
//                     "char_filter": [],
//                     "filter": ["lowercase"]
//                 }
//             }
//         }
//     },
//     "mappings": {
//         "properties": {
//             "name": {
//                 "type": "keyword",
//                 "normalizer": "lowercase"
//             },
//             ...
//         }
//     }
// }
func (es *ElasticSearch) CreateIndices(index string, indexBody string) (*elastic.IndicesCreateResult, error) {
	return es.cli.CreateIndex(index).BodyString(indexBody).Do(es.ctx)
}

// GetIndices get an indices info
func (es *ElasticSearch) GetIndices(index ...string) (map[string]*elastic.IndicesGetResponse, error) {
	return es.cli.IndexGet().Index(index...).Do(es.ctx)
}

// Index insert a data into es
func (es *ElasticSearch) Index(index string, typ string, id string, data string) (*elastic.IndexResponse, error) {
	return es.cli.Index().
		Index(index).
		Type(typ).
		Id(id).
		BodyJson(data).
		Do(es.ctx)
}

// Bulk insert some batch data into es
func (es *ElasticSearch) Bulk(index string, dataList []map[string]interface{}, delId []string) (*elastic.BulkResponse, error) {
	if len(dataList) == 0 && len(delId) == 0 {
		return nil, errors.New("invalid data")
	}
	bulkRequest := es.cli.Bulk()
	// 写入
	for _, d := range dataList {
		id, ok := d["ID"]
		if !ok {
			id, _ = uuid.GenerateUUID()
			d["ID"] = id
		} else {
			id, ok = id.(string)
			if !ok {
				id, _ = uuid.GenerateUUID()
				d["ID"] = id
			}
		}
		bulkReq := elastic.NewBulkIndexRequest().Index(index).Id(id.(string)).Doc(d)
		bulkRequest.Add(bulkReq)
	}
	// 删除
	for _, id := range delId {
		bulkReq := elastic.NewBulkDeleteRequest().Index(index).Id(id)
		bulkRequest.Add(bulkReq)
	}
	return bulkRequest.Do(es.ctx)
}

// Search ...
func (es *ElasticSearch) Search(index string, in QueryInput, from int, size int) (*elastic.SearchResult, error) {
	query := elastic.NewBoolQuery()
	queryDSL := es.genQueryDSL(query, "", in.Query, in.QueryString)
	searchService := es.cli.Search().
		Index(index).
		Query(queryDSL)
	if len(in.Ret.Includes) != 0 || len(in.Ret.Excludes) != 0 {
		fsc := elastic.NewFetchSourceContext(true)
		fsc.Include(in.Ret.Includes...)
		fsc.Exclude(in.Ret.Excludes...)
		searchService = searchService.FetchSourceContext(fsc)
	}
	if len(in.Sort) != 0 {
		sorter := es.genSorter(in.Sort)
		searchService = searchService.SortBy(sorter...)
	}
	if size != 0 {
		searchService = searchService.Size(size)
	} else {
		searchService = searchService.Size(10)
	}
	searchService = searchService.From(from).Pretty(true)
	return searchService.Do(es.ctx)
}

// GenQueryDSL 生成es原语
func (es *ElasticSearch) GenQueryDSL(in map[string]interface{}, ins string) (string, error) {
	query := elastic.NewBoolQuery()
	queryDSL := es.genQueryDSL(query, "", in, ins)
	data, err := queryDSL.Source()
	dataStr, _ := json.MarshalIndent(data, "", "    ")
	return string(dataStr), err
}

// Term 在字段"field" 里完全匹配 "value1"
// "term": {
// 	"field": "field1",
// 	"query": ["value1", "value2"]
// },
type Term struct {
	Field string        `json:"field"`
	Query []interface{} `json:"query"`
}

// Range 在字段"field" 过滤 "left", "right" 左闭右开区间
// "range": {
//    "field": "field1",
// 	"query": {
// 		"left": {
// 			"value": "left",
// 			"op": ">"
// 		},
// 		"right": {
// 			"value": "left",
// 			"op": ">"
// 		}
// 	}
// },
type Range struct {
	Field string `json:"field"`
	Query struct {
		Left struct {
			Value interface{} `json:"value"`
			Op    string      `json:"op" default:">="`
		} `json:"left"`
		Right struct {
			Value interface{} `json:"value"`
			Op    string      `json:"op" default:"<"`
		} `json:"right"`
	} `json:"query"`
}

// Exists 在字段"field" 判断 tags 是否存在
// "exists" : {
// 	"field": "field1",
// 	"query": ["tags"]
// }
type Exists struct {
	Field string `json:"field"`
	Query string `json:"query"`
}

// Match 在字段"field" 里模糊查询 "value1" 或 "value2"
// "weight" 控制对应词的匹配权重
// "type" 匹配方式 phrase
// "match": {
// 	"field": "field1",
// 	"query": [ "value1", "value2" ],
// 	"weight": [1, 2],
// 	"type": "phrase"
// }
type Match struct {
	Field  string        `json:"field"`
	Query  []interface{} `json:"query"`
	Weight []float64     `json:"weight"`
	// Type   string        `json:"type"`
}

// MultiMatch 在字段"field1", "_field2"结尾 "field3" 2倍权重 中搜索"value"
// "multi_match": {
// 	"fields": [ "field1", "*_field2", "field3^2" ],
// 	"query":  "value"
// }
type MultiMatch struct {
	Query  interface{} `json:"query"`
	Fields []string    `json:"fields"`
}

// GeoBoundingBox 找出落在指定矩形框中的点
// "geo_bounding_box": {
// 	"field": "field1",
// 	"order": "asc",
// 	"top_left": {
// 		"lat":  40.8,
// 		"lon": -74.0
// 	},
// 	"bottom_right": {
// 		"lat":  40.7,
// 		"lon": -73.0
// 	}
// }
type GeoBoundingBox struct {
	Field   string `json:"field"`
	Order   string `json:"order"`
	TopLeft struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"top_left"`
	BottomRight struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"bottom_right"`
}

// GeoDistance 找出与指定位置在给定距离内的点, 在点距离1km~2km的结果，左闭右开区间
// "geo_distance": {
// 	"field": "field1",
// 	"distance": ["1km", "2km"],
// 	"order": "asc",
// 	"location": {
// 		"lat":  40.715,
// 		"lon": -73.988
// 	}
// }
type GeoDistance struct {
	Field    string `json:"field"`
	Distance string `json:"distance"`
	Order    string `json:"order"`
	Location struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"location"`
}

// Sort 按"field1" 降序 "field2" 升序
// "sort": {
//    "field1": "desc",
//    "field2": "asc"
// }
type Sort map[string]string

// Or 组合逻辑
type Or map[string]interface{}

// And 组合逻辑
type And map[string]interface{}

// Not 组合逻辑
type Not map[string]interface{}

type Ret struct {
	Includes []string
	Excludes []string
}

// QueryInput 查询结构
type QueryInput struct {
	Query       map[string]interface{} `json:"query,omitempty"`
	QueryString string                 `json:"query_string,omitempty"`
	Ret         Ret                    `json:"ret"`
	Sort        Sort                   `json:"sort"`
	From        int                    `json:"from"`
	Size        int                    `json:"size"`
}

var logicKey = map[string]bool{
	"and": true,
	"or":  true,
	"not": true,
}

// genQueryDSL
// 在字段"field1" 里完全匹配 "value1" 或 "value2"
// {
// 	"query": {
// 		"or": {},
// 		"and": {},
// 		"not": {}
// 	},
// 	"ret": ["field1", "field2"],
// 	"sort": {
// 		"field1": "desc",
// 		"field2": "asc"
// 	},
// 	"from": 0,
// 	"size": 10
// }
func (es *ElasticSearch) genQueryDSL(query *elastic.BoolQuery, logic string, in map[string]interface{}, instr string) elastic.Query {
	if len(in) == 0 && instr == "" {
		return query
	}
	if len(in) == 0 {
		err := json.Unmarshal([]byte(instr), &in)
		if err != nil {
			return query
		}
	}
	queryFunc := query.Filter
	switch logic {
	case "and":
		queryFunc = query.Must
	case "or":
		queryFunc = query.Should
	case "not":
		queryFunc = query.MustNot
	default:
	}
	for k, v := range in {
		k = strings.ToLower(k)
		// 非嵌套结构
		if !logicKey[k] {
			switch k {
			case "term":
				nv := Term{}
				err := mapstructure.Decode(v, &nv)
				if err != nil {
					continue
				}
				// name string, value interface{}
				if len(nv.Query) == 1 {
					query = queryFunc(elastic.NewTermQuery(nv.Field, nv.Query[0]))
				} else if len(nv.Query) > 1 {
					query = queryFunc(elastic.NewTermsQuery(nv.Field, nv.Query))
				}
			case "range":
				nv := Range{}
				err := mapstructure.Decode(v, &nv)
				if err != nil {
					continue
				}
				rgQuery := elastic.NewRangeQuery(nv.Field)
				switch nv.Query.Left.Op {
				case ">":
					rgQuery = rgQuery.Gt(nv.Query.Left.Value)
				case ">=":
					rgQuery = rgQuery.Gte(nv.Query.Left.Value)
				case "<":
					rgQuery = rgQuery.Lt(nv.Query.Left.Value)
				case "<=":
					rgQuery = rgQuery.Lte(nv.Query.Left.Value)
				default:
					rgQuery = rgQuery.Gte(nv.Query.Left.Value)
				}
				switch nv.Query.Right.Op {
				case ">":
					rgQuery = rgQuery.Gt(nv.Query.Right.Value)
				case ">=":
					rgQuery = rgQuery.Gte(nv.Query.Right.Value)
				case "<":
					rgQuery = rgQuery.Lt(nv.Query.Right.Value)
				case "<=":
					rgQuery = rgQuery.Lte(nv.Query.Right.Value)
				default:
					rgQuery = rgQuery.Lt(nv.Query.Right.Value)
				}
				query = queryFunc(rgQuery)
			case "exists":
				nv := Exists{}
				err := mapstructure.Decode(v, &nv)
				if err != nil {
					continue
				}
				query = queryFunc(elastic.NewExistsQuery(nv.Field).QueryName(nv.Query))
			case "match":
				nv := Match{}
				err := mapstructure.Decode(v, &nv)
				if err != nil {
					continue
				}
				// name string, text interface{}
				for i, val := range nv.Query {
					if i >= len(nv.Weight) {
						nv.Weight = append(nv.Weight, 1.0)
					}
					query = queryFunc(elastic.NewMatchQuery(nv.Field, val).Boost(nv.Weight[i]))
				}
			case "multi_match":
				nv := MultiMatch{}
				err := mapstructure.Decode(v, &nv)
				if err != nil {
					continue
				}
				// text interface{}, fields ...string
				query = queryFunc(elastic.NewMultiMatchQuery(nv.Query, nv.Fields...))
			case "geo_bounding_box":
				nv := GeoBoundingBox{}
				err := mapstructure.Decode(v, &nv)
				if err != nil {
					continue
				}
				query = queryFunc(elastic.NewGeoBoundingBoxQuery(nv.Field).
					TopLeft(nv.TopLeft.Lon, nv.TopLeft.Lat).
					BottomRight(nv.BottomRight.Lon, nv.BottomRight.Lat))
			case "geo_distance":
				nv := GeoDistance{}
				err := mapstructure.Decode(v, &nv)
				if err != nil {
					continue
				}
				query = queryFunc(elastic.NewGeoDistanceQuery(nv.Field).
					Point(nv.Location.Lat, nv.Location.Lon).
					Distance(nv.Distance))
			default:
				continue
			}
		} else {
			var newQuery = elastic.NewBoolQuery()
			es.genQueryDSL(newQuery, k, v.(map[string]interface{}), "")
			query = queryFunc(newQuery)
		}
	}
	return query
}

func (es *ElasticSearch) genSorter(in Sort) []elastic.Sorter {
	if len(in) == 0 {
		return []elastic.Sorter{}
	}
	var sorter = make([]elastic.Sorter, 0)
	for k, v := range in {
		var sort *elastic.FieldSort
		if v == "desc" {
			sort = elastic.NewFieldSort(k).Order(false)
		} else if v == "asc" {
			sort = elastic.NewFieldSort(k).Order(true)
		}
		sorter = append(sorter, sort)
	}
	// 默认按评分降序
	sorter = append(sorter, elastic.NewScoreSort())
	return sorter
}
