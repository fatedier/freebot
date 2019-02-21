## Plugin

每一个插件都提供了一些能力，对于不同的 repo 可以启用不同的插件，并且设置不同的参数。

目前支持的插件及说明文档:

* [Assign](/plugin/assign)
* [Status](/plugin/status)
* [Merge](/plugin/merge)
* [LifeCycle](/plugin/lifecycle)
* [Notify](/plugin/notify)
* [Label](/plugin/label)

### 配置说明

插件的配置分为三个部分，示例格式如下:

```json
{
    "plugins": {
        "status": {
            "enable": true,
            "preconditions": [],
            "extra": {}
        }
    }
}
```

### enable

只在为 true 时启用。

### preconditions

前置条件，只有满足前置条件，才会继续执行后续的操作，否则不做任何操作。

格式为一个 precondition 的数组，数组中的多个 precondition 是或的关系，满足其中一个就算满足，同一个 precondition 的条件是与的关系，需要所有条件都满足才算满足。

示例:

```json
{
    "is_author": false,
    "required_roles": [],
    "required_labels": [],
    "required_label_prefix": []
}
```

* is_author: comment 的 user 是 author 自己。
* required_roles: 要求 issue 或 PR 或 comment 的 author 需要是某些指定的角色。
* required_labels: 要求 issue 或 PR 含有指定的 label。
* required_label_prefix: 要求 issue 或 PR 含有指定前缀的 label。

### extra

插件所需的额外配置，每个插件可以不同，不需要则不用配置。

具体某个插件有哪些额外配置详见插件的文档。
