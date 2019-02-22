## lgtm

通过命令或 github request review approve 的方式来给 PR 添加 `approve/` 前缀的标签。

### cmd

```
/lgtm
```

### extra

参考配置:

```json
{
    "extra": {
        "base_label_prefix": "module",
        "label_roles": {
            "module1": {
                "owner": ["user1", "user2"],
                "qa": ["user3"]
            },
            "module2": {
                "owner": ["user4"]
            }
        },
        "target_labels": [
            {
                "role": "owner",
                "target_prefix": "approve"
            }
        ]
    }
}
```
