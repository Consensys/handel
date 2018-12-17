# Examples

This example shows how to create 3 peers communicating with each other via udp transport:   
1) Open 3 terminals
2) In each terminal start:   
go run start.go -id 0 -reg local_reg.csv  
go run start.go -id 1 -reg local_reg.csv  
go run start.go -id 2 -reg local_reg.csv  
3) You should be able to see something like:  
```
msg received: Lvl 115 Org 0 1200
msg received: Lvl 253 Org 0 1200
msg received: Lvl 114 Org 0 1200
msg received: Lvl 253 Org 2 1200
msg received: Lvl 111 Org 2 1200
msg received: Lvl 117 Org 0 1200
...
```   
in the first terminal. 

