package wal

import "hash/crc32"

func (rec *Record) UpdateChecksum() {
	rec.Crc = crc32.ChecksumIEEE(rec.Data)
}

func (rec *Record) IsValid() bool {
	return rec.Crc == crc32.ChecksumIEEE(rec.Data)
}
