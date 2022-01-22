### 基本过滤器

```json
// 在字段"field" 里完全匹配 "value1" 或 "value2" 
"term": {
	"field": "field1",
	"query": ["value1", "value2"]
},

// 在字段"field" 过滤 "left", "right" op 控制比较
"range": {
   "field": "field1",
	"query": {
		"left": {
			"value": "left",
			"op": ">"
		},
		"right": {
			"value": "left",
			"op": ">"
		}
	}
},

// 在字段"field" 判断 tags 是否存在
"exists" : {
	"field": "field1",
	"query": ["tags"]
}
```



### 基本查询器

```json
// 在字段"field" 里模糊查询 "value1" 或 "value2"
// "weight" 控制对应词的匹配权重 
// "type" 匹配方式 phrase
"match": {
	"field": "field1",
	"query": [ "value1", "value2" ],
	"weight": [1, 2],
	"type": "phrase"
}
// 在字段"field1", "_field2"结尾 "field3" 2倍权重 中搜索"value"
"multi_match": {
	"fields": [ "field1", "*_field2", "field3^2" ],
	"query":  "value"
}
```



### 地理位置过滤器

```json
// 找出落在指定矩形框中的点
"geo_bounding_box": {
	"field": "field1",
	"order": "asc",
	"top_left": {
		"lat":  40.8,
		"lon": -74.0
	},
	"bottom_right": {
		"lat":  40.7,
		"lon": -73.0
	}
}

// 找出与指定位置在给定距离内的点, 在点距离1km~2km的结果，左闭右开区间
"geo_distance": {
	"field": "field1",
	"distance": ["1km", "2km"],
	"order": "asc",
	"location": {
		"lat":  40.715,
		"lon": -73.988
	}
}
```



### 排序

```json
// 按"field1" 降序 "field2" 升序
"sort": {
    "field1": "desc",
    "field2": "asc"
}
```



### 逻辑查询

```json
{
	"query": {
		"or": {},
		"and": {},
		"not": {}
	}
}
```



### 返回字段

```json
// 返回field1, field2, 从0到10条
{
    "ret": [
        "field1",
        "field2"
    ],
    "from": 0,
    "size": 10
}
```



#### 整合查询api

- 最小查询过滤单元：**基本过滤器**  **基本查询器** **地理位置过滤器**  是  **最小单元**，**内部不可以再次嵌套**
- query 包裹所有的查询和过滤
- query 里面的可以是 最小查询过滤单元 以及 他们的and、 or、 not的组合
- and、 or、 not内部可以嵌套 最小查询过滤单元 也可以是其他and、 or、 not

##### 样例

```json
{
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
}
```

