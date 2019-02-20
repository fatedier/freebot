## label

给 issue 或 PR 加上 自定义的标签，前缀和状态都是自己指定。

新增 label 和删除 label 都可以配置需要满足的条件。

### cmd

```
/{cmd} {status}
```

### extra

参考配置:

```
"label":{
    "enable": true,
    "extra":{
        "kind":{
            "add_preconditions":[
            ],
            "remove_preconditions":[
                {
                    "required_roles": ["owner"]
                }
            ],
            "labels": ["feature", "bug"]
         }
    }
}
```

上面的配置表示:

可以用 /kind feature 来打上 kind/feature 的标签,在该配置中，新增标签没有前置条件。

可以用 /remove-kind feature 来移除 kind/feature 标签， 在该配置中，前置条件为 owner。

更多信息参考 example 文件夹下的 config。
