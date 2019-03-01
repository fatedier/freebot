## lgtm

通过命令或 github request review approve 的方式来给 PR 添加 `approve/` 前缀的标签。

### cmd

```
/lgtm
/unlgtm
```

### extra

参考配置:

```json
{
    "extra": {
        "base_label_prefix": "module",
        "target_labels": [
            {
                "role": "owner",
                "target_prefix": "approve"
            }
        ]
    }
}
```
