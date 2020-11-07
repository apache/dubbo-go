# dubbo-go-cli

### 1. Problem we solved.

For the running dubbo-go server, we need a telnet-cli tool to test if the server works healthily.
The tool should support dubbo protocol, making it easy for you to define your own request pkg, get rsp struct of your server, and total costing time. 


### 2. How to get cli-tool
run in dubbo-go/tools/cli \
`$ sh build.sh`\
and you can get dubbo-go-cli 

### 3. Quick start：[example](example/README_CN.md)



### Third party dependence（temporary）

github.com/LaurenceLiZhixin/dubbo-go-hessian2 \
github.com/LaurenceLiZhixin/json-interface-parser 