package es_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	es "github.com/baiyeth/elasticsearch"
)

var queryStr = `{
    "match": {
        "field": "field1",
        "query": [
            "value1",
            "value2"
        ],
        "weight": [
            1
        ],
        "type": "phrase"
    },
    "or": {
        "range": {
            "field": "field1",
            "query": {
                "left": {
                    "value": "left"
                },
                "right": {
                    "value": "right",
                    "op": "<"
                }
            }
        },
        "terms": {
            "field": "field1",
            "query": [
                "value1"
            ]
        }
    },
    "and": {
        "or": {
            "range": {
                "field": "field1",
                "query": {
                    "left": {
                        "value": "left",
                        "op": "<="
                    },
                    "right": {
                        "value": "right",
                        "op": ">="
                    }
                }
            },
            "terms": {
                "field": "field1",
                "query": [
                    "value1",
                    "value2"
                ]
            }
        },
        "match": {
            "field": "field1",
            "query": [
                "value1",
                "value2"
            ],
            "weight": [
                1,
                2
            ],
            "type": "phrase"
        }
    },
    "not": {
        "match": {
            "field": "field1",
            "query": [
                "value1",
                "value2"
            ],
            "weight": [
                1,
                2
            ],
            "type": "phrase"
        }
    }
}`

func TestNewClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	addresses := []string{"http://127.0.0.1:9200"}
	userName := "es"
	passWord := "es"
	cli := es.NewClient(ctx, addresses, es.WithAuth(userName, passWord), es.WithMaxRetries(5), es.WithSniff(false))
	fmt.Println(cli.Ping("http://127.0.0.1:9200"))
	fmt.Println(cli.GetIndices("gate"))
}

func TestSearchQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	addresses := []string{"http://127.0.0.1:9200"}
	userName := "es"
	passWord := "es"
	cli := es.NewClient(ctx, addresses, es.WithAuth(userName, passWord), es.WithMaxRetries(5), es.WithSniff(false))
	in1 := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryStr), &in1)
	if err != nil {
		fmt.Println(err)
		return
	}
	in := es.QueryInput{
		Query: in1["query"].(map[string]interface{}),
		Ret:   es.Ret{},
		From:  0,
		Sort: es.Sort{
			"field1": "desc",
			"field2": "asc",
		},
	}
	esResult, _ := cli.Search("test_index", in, 0, 10)
	fmt.Println(esResult)
}

func TestSearchQueryString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	addresses := []string{"http://127.0.0.1:9200"}
	userName := "es"
	passWord := "es"
	cli := es.NewClient(ctx, addresses, es.WithAuth(userName, passWord), es.WithMaxRetries(5), es.WithSniff(false))
	in := es.QueryInput{
		QueryString: queryStr,
		Ret:         es.Ret{},
		From:        0,
		Sort: es.Sort{
			"field1": "desc",
			"field2": "asc",
		},
	}
	esResult, _ := cli.Search("test_index", in, 0, 10)
	fmt.Println(esResult)
}

func TestGenQueryDSL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	addresses := []string{"http://127.0.0.1:9200"}
	userName := "es"
	passWord := "es"
	cli := es.NewClient(ctx, addresses, es.WithAuth(userName, passWord), es.WithMaxRetries(5), es.WithSniff(false))
	in := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryStr), &in)
	if err != nil {
		fmt.Println(err)
		return
	}
	esResult, _ := cli.GenQueryDSL(in["query"].(map[string]interface{}), "")
	esResult2, _ := cli.GenQueryDSL(nil, queryStr)
	fmt.Println(esResult, esResult2)
}
