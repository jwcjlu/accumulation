# 流量统计功能
## How to build
```
make build
```
## How to Deploy And Run 
```
https://confluence.vrviu.com:8443/pages/viewpage.action?pageId=1629270804 
```
## 整体设计
![img.png](images/architecture.png)
## 组件设计
![img.png](images/design.png)
```azure
GsmProxy: gsm的反向代理组件，这里只要是前缀是/gsm的请求都会转发到gsm-core服务
```
```azure
BandwidthReportTask: 任务上报组件，当收到一个StrarGame事件的时候，会开启一个上报任务，该任务主要是定时从BandwidthEngine查询相应周期的流量，添加到
BandwidthReportJob组件里，多开的时候会同时运行多个task任务
```
```azure
BandwidthEngine：存储组件，定时从BandwidthCollector组件中导出流量，供BandwidthReportTask任务查询满足该任务条件（ip，port，时间段等条件）的流量
```
```azure
BandwidthCollector：流量收集组件,监听机器上的网卡，对流量进行解析供bandwidthEngine导出存储
```
```azure
BandwidthReportJob: 上报任务，一个进程只有一个job，该组件包含了RingBuf环形队列和FileStore持久化存储组件，bandwidhtReportTask收集到流量放入该job，job先添加到RingBuf
环形队列里，如果环形队列里已经满了，就持久化到FileStore里面。同时job会不断向RingBuf环形队列取数据上报到creport。当RingBuf为空的时候会检查FileStore里有没有数据并解析上报到creport
```
```azure
StoreFile: 持久化流量数据，当creport不可用以及RingBuff满的时候，新收集到的数据持久化到storeFile，当creport恢复的时候会从storeFile中load数据到RingBuff上报
```

 
## Change Logs
```

```

## About more
https://confluence.vrviu.com/pages/viewpage.action?pageId=1648861658