package es_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	es "github.com/baiyeth/elasticsearch"
)

var esStr = `{
    "query": {
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
                "query": [ "value1" ]
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
    },
    "ret": [
        "field1",
        "field2"
    ],
    "sort": {
        "field1": "desc",
        "field2": "asc"
    },
    "from": 0,
    "size": 10
}`

func TestNewClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	addresses := []string{"http://127.0.0.1:29200"}
	userName := "es"
	passWord := "es"
	cli := es.NewClient(ctx, addresses, es.WithAuth(userName, passWord), es.WithMaxRetries(5), es.WithSniff(false))
	fmt.Println(cli.Ping("http://127.0.0.1:29200"))
	fmt.Println(cli.GetIndices("gate"))
}

func TestGjson(t *testing.T) {
	t.Parallel()
	a1 := map[string]interface{}{}
	j := `{"programmers": [
    {
      "firstName": "Janet", 
      "lastName": "McLaughlin"
    }, {
      "firstName": "Elliotte", 
      "lastName": "Hunter"
    }, {
      "firstName": "Jason", 
      "lastName": "Harold"
    }
  ]
}`
	err := json.Unmarshal([]byte(j), &a1)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(a1)

	a2 := map[string]interface{}{}
	j = `{
  "name": {"first": "Tom", "last": "Anderson"},
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
    {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
    {"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
    {"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
  ]
}`
	err = json.Unmarshal([]byte(j), &a2)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(a2)

	a3 := map[string]interface{}{}
	j = `{
  "widget": {
    "debug": "on",
    "window": {
      "title": "Sample Konfabulator Widget",
      "name": "main_window",
      "width": 500,
      "height": 500
    },
    "image": { 
      "src": "Images/Sun.png",
      "hOffset": 250,
      "vOffset": 250,
      "alignment": "center"
    },
    "text": {
      "data": "Click Here",
      "size": 36,
      "style": "bold",
      "vOffset": 100,
      "alignment": "center",
      "onMouseUp": "sun1.opacity = (sun1.opacity / 100) * 90;"
    }
  }
}`
	err = json.Unmarshal([]byte(j), &a3)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(a3)

	a4 := map[string]interface{}{}
	j = `{
    "query": {
        "key": "搜索词",
        "is_term": false
	},
    "filter": {
        "or": {
			"a": 1
        },
        "and": {
			"b": 1
        },
        "not": {
			"a": 1
        }
    },
    "ret": ["field1", "field2"],
    "from": 0,
    "size": 10
}
`
	err = json.Unmarshal([]byte(j), &a4)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(a4)
}

func TestSearch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	addresses := []string{"http://127.0.0.1:29200"}
	userName := "es"
	passWord := "es"
	cli := es.NewClient(ctx, addresses, es.WithAuth(userName, passWord), es.WithMaxRetries(5), es.WithSniff(false))
	in := make(map[string]interface{})
	err := json.Unmarshal([]byte(esStr), &in)
	if err != nil {
		fmt.Println(err)
		return
	}
	esResult, _ := cli.Search("test_index", in, 0, 10)
	fmt.Println(esResult)
}

func TestGenQueryDSL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	addresses := []string{"http://127.0.0.1:29200"}
	userName := "es"
	passWord := "es"
	cli := es.NewClient(ctx, addresses, es.WithAuth(userName, passWord), es.WithMaxRetries(5), es.WithSniff(false))
	in := make(map[string]interface{})
	err := json.Unmarshal([]byte(esStr), &in)
	if err != nil {
		fmt.Println(err)
		return
	}
	esResult, _ := cli.GenQueryDSL(in["query"].(map[string]interface{}))
	fmt.Println(esResult)
}
