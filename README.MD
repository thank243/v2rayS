# v2rayS

A v2ray backend framework that can easily support many panels.

一个基于v2ray的后端框架，支持V2ay，Trojan协议，极易扩展，支持多面板对接。

由于 `v2ray-core` 暂未支持shadowsocks的单端口多用户，所以现在无法使用。

如果您喜欢本项目，可以右上角点个star+watch，持续关注本项目的进展。

教程参考：[XrayR使用教程](https://xrayr-project.github.io/XrayR-doc/)

## 免责声明

本项目只是本人个人学习开发并维护，本人不保证任何可用性，也不对使用本软件造成的任何后果负责。

## 特点

* 永久开源且免费。
* 支持V2ray，Trojan， ~~Shadowsocks~~(等待v2ray支持单端口多用户)多种协议。
* 支持单实例对接多面板、多节点，无需重复启动。
* 支持限制在线IP
* 支持节点端口级别、用户级别限速。
* 配置简单明了。
* 修改配置自动重启实例。
* 方便编译和升级，可以快速更新核心版本， 支持v2ray-core新特性。

## 功能介绍

|    功能     |       v2ray        |       trojan       |
|:---------:|:------------------:|:------------------:|
|  获取节点信息   | :white_check_mark: | :white_check_mark: |
|  获取用户信息   | :white_check_mark: | :white_check_mark: |
|  用户流量统计   | :white_check_mark: | :white_check_mark: |
|  服务器信息上报  | :white_check_mark: | :white_check_mark: |
| 自动申请tls证书 | :white_check_mark: | :white_check_mark: |
| 自动续签tls证书 | :white_check_mark: | :white_check_mark: |
|  在线人数统计   | :white_check_mark: | :white_check_mark: |
|  在线用户限制   | :white_check_mark: | :white_check_mark: |
|   审计规则    | :white_check_mark: | :white_check_mark: |
|  节点端口限速   | :white_check_mark: | :white_check_mark: |
|  按照用户限速   | :white_check_mark: | :white_check_mark: |
|  自定义DNS   | :white_check_mark: | :white_check_mark: |
|  全局用户限制   |     :pushpin:      |     :pushpin:      |

## 支持前端

|                           前端                           |       v2ray        |       trojan       |
|:------------------------------------------------------:|:------------------:|:------------------:|
|     [V2Board](https://github.com/v2board/v2board)      | :white_check_mark: | :white_check_mark: |
|  [PMPanel](https://github.com/ByteInternetHK/PMPanel)  | :white_check_mark: | :white_check_mark: |
| [ProxyPanel](https://github.com/ProxyPanel/ProxyPanel) | :white_check_mark: | :white_check_mark: |
|  [WHMCS(V2RaySocks)](https://v2raysocks.doxtex.com/)   | :white_check_mark: | :white_check_mark: |

## 配置文件及详细使用教程

参考XrayR：[详细使用教程](https://xrayr-project.github.io/XrayR-doc/)

## Thanks

* [Project X](https://github.com/XTLS/)
* [V2Fly](https://github.com/v2fly)
* [VNet-V2ray](https://github.com/ProxyPanel/VNet-V2ray)
* [Air-Universe](https://github.com/crossfw/Air-Universe)
* [XrayR](https://github.com/XrayR-project/XrayR)

## Licence

[Mozilla Public License Version 2.0](https://raw.githubusercontent.com/thank243/v2rayS/master/LICENSE)

## Telegram

[v2rayS后端讨论](https://t.me/v2rayS_chat)

[v2rayS通知](https://t.me/v2rayS_channel)

## Stargazers over time

[![Stargazers over time](https://starchart.cc/thank243/v2rayS.svg)](https://starchart.cc/thank243/v2rayS)
