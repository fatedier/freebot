## trigger

通过命令触发的方式执行外部脚本。

### cmd

```
/{cmd} {arg}
```

### extra

参考配置:

```json
{
    "extra": {
        "cmds": {
            "jenkins": {
                "command": "/home/user/scripts/jenkins.sh",
                "args": [],
                "timeout_s": 30
            }
        }
    }
}
```

用户通过 comment 触发 trigger，例如 `/jenkins app1 arg1`，freebot 会去执行 `/home/user/scripts/jenkins.sh app1 arg1`，cmd 后的参数会作为执行脚本的启动参数。

关于 issue 和 PR 的一些信息会以 json 的形式通过标准输入传入执行脚本，以换行结尾。
