package wal

import "hash/crc32"

func (rec *Record) updateChecksum() {
	rec.Crc = crc32.ChecksumIEEE(rec.Data)
}

func (rec *Record) isValid() bool {
	return rec.Crc == crc32.ChecksumIEEE(rec.Data)
}
