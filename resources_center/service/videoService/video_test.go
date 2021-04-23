package videoService

import (
	"fmt"
	"testing"
	"xgit.xiaoniangao.cn/xngo/service/resources_center/api"
)

/*
func TestDuplicateCheck(t *testing.T){
	var req api.DuplicateCheckReq
	req.Token = "FvciamOyyf-WWIgKSPAD7iuWSp__"
	req.Qetag = "FvciamOyyf-WWIgKSPAD7iuWSp__"
	resp, err := DuplicateCheck(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)
}

func TestExistResByQetag(t *testing.T) {
	exist, err := ExistResByQetag("FvciamOyyf-WWIgKSPAD7iuWSp__")
	if err != nil {
		//xlog.Error("failed to judge resource exist")
		fmt.Println("error judege exist by qetag")
	}
	fmt.Println(exist)
	return
}

*/


func TestTempVoucher(t *testing.T) {
	var req api.TempVoucherReq
	req.Token = "asfasfasf"
	info, err := GetStsVoucher()
	if err != nil {
		fmt.Println("error sts")
	}
	fmt.Printf("%s\n", info.Data.Endpoint)

}


