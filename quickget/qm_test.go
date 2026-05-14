package quickget

import (
	"fmt"
	"testing"

	"github.com/solider245/fastpve/utils"
)

var qmList = `VMID NAME                 STATUS     MEM(MB)    BOOTDISK(GB) PID       
       100 Windows10            running    16384            100.00 1539      
       101 Windows7             stopped    4096             100.00 0         
       102 iStoreOS-22.03.7-2025050912 running    2048               2.38 180304    
       103 iStoreOS-22.03.7-2025040711 running    2048               2.38 180508    
       104 iStoreOS-24.10.0-2025043010 running    2048               2.38 180627    
       105 Windows11-Sysprep    stopped    4096               0.00 0      `

func TestQMList(t *testing.T) {
	items, err := parseQMList([]byte(qmList))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		fmt.Println("item=", utils.ToString(item))
	}
}
