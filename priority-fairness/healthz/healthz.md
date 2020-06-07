## Prometheus Query:
```
sum(rate(apiserver_request_total{apiserver="kube-apiserver",verb="GET",group="",subresource="/readyz"}[1m])) by(code)

sum(rate(apiserver_request_total{apiserver="kube-apiserver",verb="GET",group="",subresource="/healthz"}[1m])) by(code)
```

```
I0519 19:41:31.532593       1 httplog.go:90] verb="GET" URI="/healthz" latency=2.055089ms resp=200 UserAgent="kube-probe/1.18+" srcIP="10.0.142.32:49780": 
I0519 19:41:36.437425       1 queueset.go:542] QS(catch-all) at r=2020-05-19 19:41:36.437407030 v=0.000000000s: immediate dispatch of request "catch-all" &request.RequestInfo{IsResourceRequest:false, Path:"/healthz", Verb:"get", APIPrefix:"", APIGroup:"", APIVersion:"", Namespace:"", Resource:"", Subresource:"", Name:"", Parts:[]string(nil)} &user.DefaultInfo{Name:"system:anonymous", UID:"", Groups:[]string{"system:unauthenticated"}, Extra:map[string][]string(nil)}, qs will have 1 executing
I0519 19:41:36.437502       1 queueset.go:337] QS(catch-all): Dispatching request &request.RequestInfo{IsResourceRequest:false, Path:"/healthz", Verb:"get", APIPrefix:"", APIGroup:"", APIVersion:"", Namespace:"", Resource:"", Subresource:"", Name:"", Parts:[]string(nil)} &user.DefaultInfo{Name:"system:anonymous", UID:"", Groups:[]string{"system:unauthenticated"}, Extra:map[string][]string(nil)} from its queue
```

```
I0519 20:40:28.516260       1 handler.go:153] kube-aggregator: GET "/readyz" satisfied by nonGoRestful
I0519 20:40:28.516279       1 pathrecorder.go:240] kube-aggregator: "/readyz" satisfied by exact match
I0519 20:40:28.518457       1 queueset.go:655] QS(catch-all) at r=2020-05-19 20:40:28.518447534 v=0.000000000s: request &request.RequestInfo{IsResourceRequest:false, Path:"/readyz", Verb:"get", APIPrefix:"", APIGroup:"", APIVersion:"", Namespace:"", Resource:"", Subresource:"", Name:"", Parts:[]string(nil)} &user.DefaultInfo{Name:"system:anonymous", UID:"", Groups:[]string{"system:unauthenticated"}, Extra:map[string][]string(nil)} finished, qs will have 0 executing
I0519 20:40:28.518535       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.549306ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.21.194:60839": 
I0519 20:40:28.581837       1 queueset.go:542] QS(catch-all) at r=2020-05-19 20:40:28.581803911 v=0.000000000s: immediate dispatch of request "catch-all" &request.RequestInfo{IsResourceRequest:false, Path:"/readyz", Verb:"get", APIPrefix:"", APIGroup:"", APIVersion:"", Namespace:"", Resource:"", Subresource:"", Name:"", Parts:[]string(nil)} &user.DefaultInfo{Name:"system:anonymous", UID:"", Groups:[]string{"system:unauthenticated"}, Extra:map[string][]string(nil)}, qs will have 1 executing
I0519 20:40:28.581921       1 queueset.go:337] QS(catch-all): Dispatching request &request.RequestInfo{IsResourceRequest:false, Path:"/readyz", Verb:"get", APIPrefix:"", APIGroup:"", APIVersion:"", Namespace:"", Resource:"", Subresource:"", Name:"", Parts:[]string(nil)} &user.DefaultInfo{Name:"system:anonymous", UID:"", Groups:[]string{"system:unauthenticated"}, Extra:map[string][]string(nil)} from its queue
```

```
&request.RequestInfo{
    IsResourceRequest:false, 
    Path:"/healthz", 
    Verb:"get", 
    APIPrefix:"", 
    APIGroup:"", 
    APIVersion:"", 
    Namespace:"", 
    Resource:"", 
    Subresource:"", 
    Name:"", 
    Parts:[]string(nil)
} 

&user.DefaultInfo{
    Name:"system:anonymous", 
    UID:"", 
    Groups:[]string{
        "system:unauthenticated"
    }, 
    Extra:map[string][]string(nil)
}
```

```
&request.RequestInfo{
    IsResourceRequest:false, 
    Path:"/readyz", 
    Verb:"get", 
    APIPrefix:"", 
    APIGroup:"", 
    APIVersion:"", 
    Namespace:"", 
    Resource:"", 
    Subresource:"", 
    Name:"", 
    Parts:[]string(nil)
} 

&user.DefaultInfo{
    Name:"system:anonymous", 
    UID:"", Groups:[]string{
        "system:unauthenticated"
    }, 
    Extra:map[string][]string(nil)
}
```

```
I0519 21:02:19.167224       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.518369ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.21.194:56765": 
I0519 21:02:19.179642       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.15685ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.177.26:12996": 
I0519 21:02:19.292362       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.772816ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.177.26:34559": 
I0519 21:02:19.399174       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.493861ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.177.26:25519": 
I0519 21:02:19.532192       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.328375ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.62.129:6785": 
I0519 21:02:19.567682       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.02946ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.74.157:2684": 
I0519 21:02:19.614329       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.725989ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.1.39:13231": 
I0519 21:02:19.650916       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.250872ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.193.224:16400": 
I0519 21:02:19.678901       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.390884ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.213.162:13353": 
I0519 21:02:19.727744       1 httplog.go:90] verb="GET" URI="/readyz" latency=1.908976ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.74.157:23029": 
I0519 21:02:19.737180       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.509745ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.213.162:37256": 
I0519 21:02:19.762843       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.149537ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.1.39:53218": 
I0519 21:02:19.807516       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.208501ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.142.204:41089": 
I0519 21:02:19.964971       1 httplog.go:90] verb="GET" URI="/readyz" latency=2.297911ms resp=200 UserAgent="ELB-HealthChecker/2.0" srcIP="10.0.177.26:20543": 
```
