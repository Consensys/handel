open 3 terminals  
cd /github.com/ConsenSys/handel/network/libp2p/examples  
1)  
terminal1: go run start.go -id 0 -reg 3  
terminal2: go run start.go -id 1 -reg 3  
terminal3: go run start.go -id 2 -reg 3  
  
after around 30s the program deadlocs.  
  
2)  
go to: handel/network/libp2p/net.go

change 
```go
 const protocol = "/handel/1.0.0"
```
to 
```go
 const protocol = ""
```
go to point 2  
no deadloc anymore


