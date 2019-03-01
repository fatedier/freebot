## notify

用于通知某些 Event 的内容。

### cmd

```
/ping {user} {message}
```

### extra

参考配置:

```json
{
    "extra": {
        "user_notify_confs": {
            "user1": {
                "slack": {
                    "url": "xxx",
                    "channel": "@user1"
                }
            },
            "user2": {
                "slack": {
                    "url": "xxx",
                    "channel": "@user2"
                }
            }
        },
        "ping": {
            "preconditions": []
        },
        "events": {
            "check_run_complete": {
                "default_user": "user1",
                "users": ["user2"]
            },
            "check_suite_complete": {
                "users": ["user1", "user2"]
            }
        },
        "crontab": {
            "wait_review_pull_request":{
                "disable": false,
                "preconditions": [],
                "sender_to_users": [],  # 为空则发给所有 request reviewers
                "job": "0 30 * * * *"   # 每半个小时通知一次
             }
         }
    }
}
```

上面的配置表示有两个用户配置了 slack 的通知方式。

对于 `check_run_complete` 和 `check_suite_complete` 两个事件会通过配置的通知方式推送给指定的用户，如果没有在 users 中配置，则会推送给 `default_user`。

### ping

可以用过 `/ping {user} {message}` 的 comment 将其所属的 issue 或 PR 内容及消息通知给指定的用户。
