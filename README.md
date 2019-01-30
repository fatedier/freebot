# freebot

freebot 是一个帮助团队基于 github 的 issue 和 pull request 进行工作流管理的 bot。

支持通过 comment 命令的方式执行一些经过前置检查的操作以及一些自动化的流程。

## 目录

<!-- vim-markdown-toc GFM -->

* [简单示例](#简单示例)
* [功能](#功能)
    * [插件](#插件)
    * [别名](#别名)
    * [角色](#角色)

<!-- vim-markdown-toc -->

### 简单示例

通过示例的 [freebot.conf](./example/freebot.conf) 快速尝试。

根据自己的需要修改 owner, repo 和 plugin 的相关配置。

执行 `./freebot -c ./freebot.conf`

在 github 上配置 repo 的 webhook 地址为 freebot 的监听地址。

示例配置的简单工作流说明:

1. 开发通过 `/status wip` 给 PR 加上 `status/wip` 的标签。
2. 开发完成，开发执行 `/status wait-review` 将状态修改为待 review 且 `/cc @user1` 将 user1 指定为 reviewer。
3. user1 如果 review 未通过，可以将状态修改为 `request-changes`。
4. user1 如果 review 通过，`/status approved` 将状态修改为 `approved`。
5. 开发执行 `/status testing` 并通过 `/cc @user2` 抄送 QA。
6. QA 收到通知，开始测试，如果测试不通过，回到 `request-changes`，测试通过，则 `/status merge-ready`。
7. 开发通过 `/merge` 将代码合并，进行后续的上线操作。

### 功能

#### 插件

每一个插件提供了一些基础的能力，通过为每一个 repo 进行插件配置从而实现定制化的团队工作流。

[插件详细说明](./plugin/README.md)

#### 别名

在 github 的 comment 中敲命令，没有自动补全是一件很繁琐的事，通过设置别名来简化这一操作。

```
{
    "alias": {
        "cmds": {
            "s": "status"
        },
        "labels": {
            "wr": "wait-review"
        },
        "users": {
            "aaa": "bbb"
        }
    }
}
```

可以给命令，标签和用户分别设置别名。

原先需要添加评论 `/status wait-review`，使用别名后只需要输入 `/s wr` 即可。

#### 角色

目前有三种角色，owner, QA, normal。

owner: 管理者。
QA: 质量管理人员。
normal: 普通开发者。

可以为每一个 repo 分别指定各个角色的 user，之后可以在插件中使用，例如限制只有 owner 才能执行某个插件的操作。

示例:

```json
{
    "roles": {
        "owner": ["user1", "user2"],
        "qa": ["qa1"]
    }
}
```
