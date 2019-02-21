## module

根据 Pull Request 的修改的文件自动加上 `module/` 前缀的标签。

### cmd

无。

### extra

参考配置:

```json
{
    "extra": {
        "label_prefix": "module",
        "file_prefix_map": {
            "dev/foo/": "foo",
            "dev/bar/": "bar",
            "": "all"
        }
    }
}
```

上述配置表示根据文件前缀的匹配，会加上 `module/` 开头的标签。
