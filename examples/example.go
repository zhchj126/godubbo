package main

import (
    "fmt"
    zk  "github.com/zhchj126/godubbo"
    "reflect"
)

//zh.springboot.service.Address
type Address struct {
    City, Country string
}

//zh.springboot.service.Person
type Person struct {
    Age     int32
    Name    string
    Address Address
}

func main() {
    var tcpAddress = "127.0.0.1:18080";
    javaClassMap := map[string]reflect.Type{
        "zh.springboot.service.Address" : reflect.TypeOf(Address{}),
        "zh.springboot.service.Person" :reflect.TypeOf(Person{}),
    }

    dubboCtx := &zk.DubboCtx{
        DubboVer: "2.5.3.6",
        Service :"zh.springboot.service.ClassService",
        Method : "getPersons2",
        Version:"go",
        Timeout:5000,
        JavaClassMap: javaClassMap,
        Return : reflect.TypeOf([]Person{}),
    }

    //args map; key: arg_type value:arg_value
    args := make(map[string]interface{})
    var srcArray []Person = make([]Person, 2)
    address := Address{"上海", "中国"}
    address2 := Address{"shanghai", "china"}
    person := Person{12, "姓名", address}
    person2 := Person{11, "xm", address2}
    srcArray[0] = person
    srcArray[1] = person2
    args["java.util.List"] = srcArray

    dubboCtx.Args = args
    fmt.Println("send ; %s", args)

    ret, _ := zk.SendHession(tcpAddress, dubboCtx)
    fmt.Println("ret ; %s", ret)

}



