package addrs

import (
	"fmt"
	"io"

	"errors"

	"github.com/MDLlife/teller/src/util"
	"github.com/boltdb/bolt"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/cipher"
)

const skyBucketKey = "used_sky_address"

// NewSKYAddrs returns an Addrs loaded with SKY addresses
func NewSKYAddrs(log logrus.FieldLogger, db *bolt.DB, addrsReader io.Reader) (*Addrs, error) {
	loader, err := loadSKYAddresses(addrsReader)
	if err != nil {
		log.WithError(err).Error("Load deposit skycoin address list failed")
		return nil, err
	}
	return NewAddrs(log, db, loader, skyBucketKey)
}

func loadSKYAddresses(addrsReader io.Reader) (addrs []string, err error) {
	addrs, err = util.ReadLines(addrsReader)
	if err != nil {
		return nil, fmt.Errorf("Decode loaded address failed: %v", err)
	}

	if err := verifySKYAddresses(addrs); err != nil {
		return nil, err
	}

	return addrs, nil
}

// func validSKYCheckSum(s string) error {
// 	if len(s) != 34 && len(s) != 35 {
// 		fmt.Println("validSKYCheckSum, ", len(s))
// 		return errors.New("Invalid address length")
// 	}
// 	return nil
// }

func verifySKYAddresses(addrs []string) error {
	if len(addrs) == 0 {
		return errors.New("No SKY addresses")
	}

	addrMap := make(map[string]struct{}, len(addrs))

	for _, addr := range addrs {
		if _, ok := addrMap[addr]; ok {
			return fmt.Errorf("Duplicate deposit address `%s`", addr)
		}

		// if err := validSKYCheckSum(addr); err != nil {
		// 	return fmt.Errorf("Invalid deposit address `%s`: %v", addr, err)
		// }

		_, err := cipher.DecodeBase58Address(addr)
		if err != nil {
			return fmt.Errorf("Invalid deposit address `%s`: %v", addr, err)
		}

		addrMap[addr] = struct{}{}
	}

	return nil
}
