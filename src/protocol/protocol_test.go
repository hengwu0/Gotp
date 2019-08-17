package protocol

import "testing"

func TestProtocol(t *testing.T) {
	if BytesToInt64(Int64ToBytes(56)) == 57 {
		t.Errorf("IntToBytes and BytesToInt failed! %d\n", BytesToInt(IntToBytes(56)))
	}
}

func TestProtocol2(t *testing.T) {
	switch "Send5" {
	case "Recv":
		t.Errorf("Recv ok!\n")
	case "Send":
		t.Errorf("Send ok!\n")
	}
}

func TestProtocol3(t *testing.T) {
	t.Errorf("Recv ok!\n", Enpack("Send", 54, "sadf"))
}
