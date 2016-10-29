package go_zookeeper_dubbo

import (
    "net"
    "fmt"
    . "github.com/zhchj126/gohessian"
    "bytes"
    "strings"
    "reflect"
    "errors"
    "time"
    "strconv"
)

const (
    DEFAULT_LEN = 8388608    // 8 * 1024 * 1024 default body max length
    PACKET_DEFAULT_LEN = 131400    // 128 * 1024 default one buf max length

    Response_OK byte = 20
    Response_CLIENT_TIMEOUT byte = 30
    Response_SERVER_TIMEOUT byte = 31
    Response_BAD_REQUEST byte = 40
    Response_BAD_RESPONSE byte = 50
    Response_SERVICE_NOT_FOUND byte = 60
    Response_SERVICE_ERROR byte = 70
    Response_SERVER_ERROR byte = 80
    Response_CLIENT_ERROR byte = 90

    RESPONSE_WITH_EXCEPTION int32 = 0
    RESPONSE_VALUE int32 = 1
    RESPONSE_NULL_VALUE int32 = 2
)

type (
    DubboCtx struct {
        DubboVer     string
        Service      string
        Method       string
        Version      string
        Args         map[string]interface{}
        Timeout      int
        JavaClassMap map[string]reflect.Type
        Return       reflect.Type
    }
)

func SendHession(tcpaddr string, service *DubboCtx) (interface{}, error) {

    var stream = service;
    //封装hessian over tcp 请求

    path := stream.Service
    method := stream.Method
    args := service.Args
    decodeClsMap := service.JavaClassMap
    retType := service.Return

    //classMap 反转
    //goClassName---JavaClassName 用于encode
    //JavaClassName---goClassName 用于decode
    var encoderClsMap map[string]string = make(map[string]string)
    for k, v := range decodeClsMap {
        encoderClsMap[v.Name()] = k;
    }
    bodyBytes, _ := bufferBody(service.DubboVer, path, method, service.Version, args, service.Timeout, encoderClsMap);
    headBytes, _ := bufferHeader(len(bodyBytes));

    headBytes = append(headBytes, bodyBytes...)

    tcpAddr, err := net.ResolveTCPAddr("tcp4", tcpaddr)
    if (checkError(err, "ResolveTCPAddr") == false) {
        return nil, err
    }
    conn, err := net.DialTCP("tcp", nil, tcpAddr)
    if (checkError(err, "DialTCP") == false) {
        return nil, err
    }
    defer conn.Close()
    //conn write
    writeLength, err := conn.Write(headBytes)
    if (checkError(err, "chatSend") == false) {
        return nil, err
    }
    fmt.Println(conn.RemoteAddr().String(), "send lens: ", writeLength)
    //conn read
    buf, retLengh, err := handleConnection(conn, service.Timeout)
    //retLengh, err := conn.Read(buf)
    if (checkError(err, "chatRead") == false) {
        return nil, err
    }
    fmt.Println(conn.RemoteAddr().String(), "received lens: ", retLengh)
    ret, err := decodeRespon(buf[0:retLengh], decodeClsMap);
    if (checkError(err, "recv") == false) {
        return nil, err
    }
    retObject := retReflect(ret, retType);
    return retObject, err

}
func handleConnection(conn net.Conn, timeout int) ([]byte, int, error) {
    ret := make([]byte, DEFAULT_LEN)
    buffer := make([]byte, PACKET_DEFAULT_LEN)
    offset := 0
    var err error =nil

    var bodyLength int
    var errCh chan int = make(chan int, 1)
    var ch chan int = make(chan int, 1)

    go  func (){for {
        n, err := conn.Read(buffer)
        if err != nil {
            errCh <-1
            //return nil, 0, err
        }
        copy(ret[offset:], buffer[:n])
        offset = offset + n

        if (offset >= 16) {
            bodyLength = int(ret[12]) << 24 + int(ret[13]) << 16 + int(ret[14]) << 8 + int(ret[15])
        }
        if (offset >= bodyLength + 16) {
            ch<-1
            //return ret, offset, nil
        }
    }}()
    select {
    case <-ch: //返回结果
    case <-errCh:
    case <-time.After( time.Millisecond*time.Duration(timeout)): //超时5s
        err=errors.New("timeout after "+strconv.Itoa(timeout) +" ms")
    }
    return ret, offset, err

}

//reflect return value
func retReflect(ret interface{}, retType reflect.Type) (interface{}) {
    if ret == nil {
        return nil
    }
    typ := retType

    switch typ.Kind() {
    case reflect.String:
        return ret.(string)
    case reflect.Int32:
        return ret.(int32)
    case reflect.Int64:
        return ret.(int64)
    case reflect.Slice, reflect.Array:
        ind := ret.([]interface{})
        //retArray := reflect.MakeSlice(reflect.SliceOf(typ), len(ind), len(ind)).Interface().([][]interface{})
        //for i:=0;i<len(ind);i++{
        //	retArray[0][i]=ind[i].(reflect.Value).Elem().Interface()
        //}
        var retArray []interface{} = make([]interface{}, len(ind))
        for i := 0; i < len(ind); i++ {
            retArray[i] = ind[i].(reflect.Value).Elem().Interface()
        }
        return retArray
    case reflect.Float32:
        return ret.(float32)
    case reflect.Float64:
        return ret.(float64)
    //case reflect.Uint8:
    case reflect.Map:
        ind := ret.(map[interface{}]interface{})
        var retMap map[interface{}]interface{} = make(map[interface{}]interface{}, len(ind))
        for k, v := range ind {
            vv := v.(reflect.Value).Elem().Interface()
            retMap[k] = vv
        }
        return retMap
    case reflect.Bool:
        return ret.(bool)
    case reflect.Struct:
        ind := ret.(interface{})
        return ind
    }
    return nil
}

//hessian decode respon
func decodeRespon(buff []byte, clsMap map[string]reflect.Type) (interface{}, error) {
    length := len(buff)

    if (buff[3] != Response_OK) {
        fmt.Println("Response not OK", string(buff[18:length - 1]))
        return nil, errors.New("Response not OK")
    }
    br := bytes.NewBuffer(buff[16:length])
    decoder := NewDecoder(br, clsMap)

    aaa, _ := decoder.ReadObject()
    switch aaa {
    case RESPONSE_WITH_EXCEPTION:
        return decoder.ReadObject()
    case RESPONSE_VALUE:
        return decoder.ReadObject()
    case RESPONSE_NULL_VALUE:
        fmt.Println("Received null")
        return nil, errors.New("Received null")
    }
    return nil, nil;

}
//creat request buffer body
func bufferBody(dver string, path string, method string, version string, args map[string]interface{}, timeout int, clsMap map[string]string) (body []byte, err error) {

    //模拟dubbo的请求
    buff := bytes.NewBuffer(nil)
    encoder := NewEncoder(buff, clsMap)

    encoder.WriteObject(dver)
    encoder.WriteObject(path)
    encoder.WriteObject(version)
    encoder.WriteObject(method)

    var types string = getTypes(args)
    encoder.WriteObject(types)//"Ljava/lang/Integer;"

    // 遍历map
    for _, v := range args {
        encoder.WriteObject(v)
    }

    implicitArgs := make(map[string]string)
    implicitArgs["path"] = path
    implicitArgs["interface"] = path
    implicitArgs["version"] = version
    implicitArgs["timeout"] = strconv.Itoa(timeout)

    encoder.WriteObject(implicitArgs)

    return buff.Bytes(), nil
}
func getTypes(args map[string]interface{}) string {

    var types string = "";
    // 初始化 + 赋值, 基本类型的序列化tag
    typeRef := map[string]string{
        "boolean": "Z",
        "int": "I",
        "short": "S",
        "long"   : "J",
        "double": "D",
        "float": "F",
    }

    // 遍历map
    for k, _ := range args {
        if (!strings.Contains(k, ".")) {
            types += typeRef[k]
        } else {
            types = types + "L" + strings.Replace(k, ".", "/", -1) + ";"
        }
    }

    return types
}

//creat request buffer head
func bufferHeader(len int) ([]byte, error) {
    // magic code
    var head = []byte{0xda, 0xbb, 0xc2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0};
    var i = 15;
    if (len > DEFAULT_LEN) {
        fmt.Printf(`Data length too large: ${length}, max payload: ${DEFAULT_LEN}`);
    }

    for len >= 256 {
        head[i] = byte(len % 256);
        len >>= 8;
        i--
    }
    head[i] = byte(len % 256);
    return head, nil;

}
//error check
func checkError(err error, info string) (res bool) {
    if (err != nil) {
        fmt.Println(info + "  " + err.Error())
        return false
    }
    return true
}

