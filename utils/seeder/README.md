# Work Order System - Data Seeder

Utilitas ini digunakan untuk menghasilkan data dummy untuk sistem Work Order. Seeder akan membuat pengguna, work order, riwayat status, dan progress work order dengan data acak.

## Fitur

- Membuat pengguna dengan peran Production Manager dan Operator
- Membuat work order dengan status acak (Pending, In Progress, Completed)
- Membuat riwayat status untuk setiap work order
- Membuat entri progress untuk work order dengan status In Progress atau Completed
- Semua data dibuat dengan tanggal antara 1 Februari 2025 dan 28 Februari 2025

## Cara Penggunaan

1. Pastikan database PostgreSQL sudah berjalan dan konfigurasi di file `.env` sudah benar
2. Jalankan seeder dengan perintah:

```bash
cd backend/utils/seeder
go run seeder.go
```

## Konfigurasi

Anda dapat mengubah jumlah data yang dihasilkan dengan mengedit konstanta berikut di file `seeder.go`:

```go
const (
    UserCount        = 10    // Jumlah operator yang dibuat
    WorkOrderCount   = 200   // Jumlah work order yang dibuat
    ProgressCount    = 500   // Jumlah maksimum entri progress
)
```

## Data yang Dihasilkan

### Pengguna

- 1 Production Manager dengan username `manager` dan password `password`
- 10 Operator dengan username `operator1`, `operator2`, dst. dan password `password`

### Work Order

- 200 work order dengan status acak
- Setiap work order memiliki riwayat status
- Work order dengan status In Progress atau Completed memiliki entri progress

## Catatan

- Seeder akan menghapus semua data yang ada di database sebelum membuat data baru
- Pastikan Anda tidak menjalankan seeder di lingkungan produksi
