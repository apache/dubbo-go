go run . -host="localhost" -port=20000 -protocol="dubbo"\
 -interface="com.ikurento.user.UserProvider"\
 -method="GetUser" -sendObj="./userCall.json"\
 -recvObj="./user.json"

