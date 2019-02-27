## status

给 issue 或 PR 加上 `status/` 前缀的标签，用于指定状态。

状态是唯一的，加上某个状态，会删除之前的状态。每一种状态都可以设置前置条件，在满足某些条件下才能被更改为指定状态。

有两种触发方式，一种是通过 comment 主动触发， 一种是通过 event trigger 满足某些条件的情况下被动修改为某个状态。

### cmd

```
/status {status}
```

### extra

参考配置:

```json
{
    "extra": {
        "events_trigger": {
            "pull_request/opened": [{
                "status": "wip",
                "preconditions": []
            }],
            "pull_request_review/submitted/approved": [{
                "status": "approved",
                "preconditions": [{
                    "required_roles": ["owner"]
                }]
            },{
                "status": merge-ready",
                "preconditions": [{
                    "required_roles": ["qa"]
                }]
            }]
        },

        "label_precondition": {
            "wip": [],
            "wait-review": [],
            "request-changes": [],
            "approved": [{
                "required_roles": ["owner"]
            }],
            "testing": [{
                "required_labels": ["status/approved"]
            }],
            "merge-ready": [
                {
                    "required_roles": ["owner"]
                },
                {
                    "required_roles": ["qa"],
                    "required_labels": ["status/testing"]
                }
            ]
        }
    }
}
```

#### label_precondition

上面的配置表示 wip, wait-review, request-changes 这三种状态任何人可以添加。

approved 状态只能由 owner 修改。

testing 状态需要处于 approved 状态才能修改。

merge-ready 状态有两种情况都可以，一种是 owner 可以直接修改，另外一种是 QA 可以修改且需要处于 testing 状态。

#### events_trigger

目前支持的事件

```
pull_request/opened
pull_request/synchronize
pull_request/labeled
pull_request/unlabeled
pull_request_review/submitted/approved
pull_request_review/submitted/commented
pull_request_review/submitted/changes_requested
```

每一个事件可以配置多个 status 以及其对应的前置条件。

当事件触发时，会按照配置的顺序进行前置条件检查，若满足条件，则会将状态修改为指定的 status。
