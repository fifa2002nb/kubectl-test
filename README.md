# kubectl-test
实现的一个kubectl插件，用于目标pod的namespaces（IPC, PID, NETWORK, USERNS）的非侵入式attach。

# 原理
1. 本地执行插件kubectl test cmd --configure /etc/conf.ini；
2. test插件通过k8s client在目标pod所在node上起一个agentPod, 插件agentPod与目标Pod同namespace；
3. 本地test插件与目标pod所在node的agentPod建立spdy协议通信；
4. agentPod获得test插件http请求后，使用docker api在node上起一个用于调试的container并让test插件执行用户attach到该container, 调试container与目标pod namespace相同；
5. test插件用户detach调试container后，会自动做清扫工作，agentPod将调试container删除，agentPod被k8s删除；

# 执行
```bash
mv kubectl-test /usr/bin
mv conf.ini /etc/conf.ini
kubectl test cmd --configure /etc/conf.ini
```
