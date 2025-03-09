package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/dawamr/work-order-system-go/config"
	"github.com/dawamr/work-order-system-go/database"
	"github.com/dawamr/work-order-system-go/models"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Jumlah data yang akan dibuat
	UserCount        = 10
	WorkOrderCount   = 2000
	ProgressCount    = 200
	StatusHistoryMin = 1
	StatusHistoryMax = 3
)

// Rentang tanggal untuk data
var (
	startDate = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	endDate   = time.Date(2025, 2, 28, 23, 59, 59, 0, time.UTC)
)

// Daftar nama produk untuk data dummy
var productNames = []string{
	"Smartphone X1", "Laptop Pro", "Wireless Earbuds", "Smart Watch", "Tablet Ultra",
	"Desktop PC", "Gaming Console", "Bluetooth Speaker", "Wireless Mouse", "Mechanical Keyboard",
	"LED Monitor", "External SSD", "Power Bank", "Wireless Charger", "USB-C Hub",
	"Router", "Security Camera", "Smart Bulb", "Drone", "Action Camera",
}

// Daftar deskripsi progress untuk data dummy
var progressDescriptions = []string{
	"Memulai proses produksi",
	"Menyiapkan bahan baku",
	"Melakukan perakitan komponen",
	"Melakukan pengujian awal",
	"Menyelesaikan 25% produksi",
	"Menyelesaikan 50% produksi",
	"Menyelesaikan 75% produksi",
	"Melakukan pengujian kualitas",
	"Melakukan perbaikan minor",
	"Melakukan pengemasan produk",
	"Mempersiapkan pengiriman",
	"Menyelesaikan dokumentasi produksi",
	"Melakukan inspeksi akhir",
	"Menyelesaikan seluruh produksi",
	"Menyerahkan produk ke bagian QA",
}

func main() {
	// Load configuration
	config.LoadConfig()

	// Connect to database
	database.ConnectDB()

	// Migrate database
	database.MigrateDB()

	// Seed data
	seedUsers()
	seedWorkOrders()

	fmt.Println("Seeding completed successfully!")
}

// Menghasilkan tanggal acak dalam rentang yang ditentukan
func randomDate(start, end time.Time) time.Time {
	delta := end.Unix() - start.Unix()
	sec := rand.Int63n(delta) + start.Unix()
	return time.Unix(sec, 0)
}

// Menghasilkan password hash
func hashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	return string(hashedPassword)
}

// Seed data pengguna
func seedUsers() {
	fmt.Println("Seeding users...")

	// Hapus data pengguna yang ada
	database.DB.Unscoped().Where("1 = 1").Delete(&models.User{})

	// Buat Production Manager
	productionManager := models.User{
		Username: "manager",
		Password: "password",
		Role:     models.RoleProductionManager,
	}
	database.DB.Create(&productionManager)
	fmt.Println("Created Production Manager: manager / password")

	// Buat Operators
	for i := 1; i <= UserCount; i++ {
		operator := models.User{
			Username: fmt.Sprintf("operator%d", i),
			Password: "password",
			Role:     models.RoleOperator,
		}
		database.DB.Create(&operator)
	}
	fmt.Printf("Created %d Operators\n", UserCount)
}

// Seed data work order
func seedWorkOrders() {
	fmt.Println("Seeding work orders...")

	// Hapus data work order yang ada
	database.DB.Unscoped().Where("1 = 1").Delete(&models.WorkOrderStatusHistory{})
	database.DB.Unscoped().Where("1 = 1").Delete(&models.WorkOrderProgress{})
	database.DB.Unscoped().Where("1 = 1").Delete(&models.WorkOrder{})

	// Dapatkan semua operator
	var operators []models.User
	database.DB.Where("role = ?", models.RoleOperator).Find(&operators)

	if len(operators) == 0 {
		log.Fatal("No operators found. Please seed users first.")
	}

	// Buat work orders
	for i := 1; i <= WorkOrderCount; i++ {
		// Pilih operator secara acak
		operator := operators[rand.Intn(len(operators))]

		// Tentukan tanggal pembuatan dan deadline
		createdAt := randomDate(startDate, endDate)
		productionDeadline := createdAt.Add(time.Hour * 24 * time.Duration(rand.Intn(14)+1)) // 1-14 hari setelah dibuat

		// Tentukan status secara acak
		statusOptions := []models.WorkOrderStatus{
			models.StatusPending,
			models.StatusInProgress,
			models.StatusCompleted,
		}
		status := statusOptions[rand.Intn(len(statusOptions))]

		// Buat work order number
		workOrderNumber := fmt.Sprintf("WO-%s-%03d", createdAt.Format("20060102"), i%999+1)

		// Pilih nama produk secara acak
		productName := productNames[rand.Intn(len(productNames))]

		// Buat work order
		targetQuantity := rand.Intn(100) + 1 // 1-100
		workOrder := models.WorkOrder{
			WorkOrderNumber:    workOrderNumber,
			ProductName:        productName,
			TargetQuantity:     targetQuantity,
			Quantity:           rand.Intn(targetQuantity),
			ProductionDeadline: productionDeadline,
			Status:             status,
			OperatorID:         operator.ID,
			CreatedAt:          createdAt,
			UpdatedAt:          createdAt,
		}

		// Simpan work order
		result := database.DB.Create(&workOrder)
		if result.Error != nil {
			log.Fatalf("Failed to create work order: %v", result.Error)
		}

		// Buat riwayat status
		seedWorkOrderStatusHistory(workOrder)

		// Jika status in progress atau completed, buat progress entries
		if status == models.StatusInProgress || status == models.StatusCompleted {
			seedWorkOrderProgress(workOrder)
		}
	}

	fmt.Printf("Created %d Work Orders\n", WorkOrderCount)
}

// Seed data riwayat status work order
func seedWorkOrderStatusHistory(workOrder models.WorkOrder) {
	// Selalu buat status awal "pending"
	pendingHistory := models.WorkOrderStatusHistory{
		WorkOrderID: workOrder.ID,
		Status:      models.StatusPending,
		Quantity:    workOrder.Quantity,
		CreatedAt:   workOrder.CreatedAt,
		UpdatedAt:   workOrder.CreatedAt,
	}
	database.DB.Create(&pendingHistory)

	// Jika status in progress atau completed, tambahkan riwayat in progress
	if workOrder.Status == models.StatusInProgress || workOrder.Status == models.StatusCompleted {
		inProgressDate := workOrder.CreatedAt.Add(time.Hour * 24 * time.Duration(rand.Intn(3)+1)) // 1-3 hari setelah dibuat
		inProgressHistory := models.WorkOrderStatusHistory{
			WorkOrderID: workOrder.ID,
			Status:      models.StatusInProgress,
			Quantity:    workOrder.Quantity,
			CreatedAt:   inProgressDate,
			UpdatedAt:   inProgressDate,
		}
		database.DB.Create(&inProgressHistory)
	}

	// Jika status completed, tambahkan riwayat completed
	if workOrder.Status == models.StatusCompleted {
		completedDate := workOrder.CreatedAt.Add(time.Hour * 24 * time.Duration(rand.Intn(5)+4)) // 4-8 hari setelah dibuat
		completedHistory := models.WorkOrderStatusHistory{
			WorkOrderID: workOrder.ID,
			Status:      models.StatusCompleted,
			Quantity:    workOrder.Quantity,
			CreatedAt:   completedDate,
			UpdatedAt:   completedDate,
		}
		database.DB.Create(&completedHistory)
	}
}

// Seed data progress work order
func seedWorkOrderProgress(workOrder models.WorkOrder) {
	// Tentukan jumlah entri progress (1-5)
	progressEntries := rand.Intn(5) + 1

	for i := 0; i < progressEntries; i++ {
		// Tentukan tanggal progress
		progressDate := workOrder.CreatedAt.Add(time.Hour * 24 * time.Duration(rand.Intn(5)+1)) // 1-5 hari setelah dibuat

		// Jika status completed, pastikan tanggal progress sebelum tanggal completed
		if workOrder.Status == models.StatusCompleted {
			// Dapatkan tanggal completed dari riwayat status
			var completedHistory models.WorkOrderStatusHistory
			database.DB.Where("work_order_id = ? AND status = ?", workOrder.ID, models.StatusCompleted).First(&completedHistory)

			if completedHistory.ID != 0 && progressDate.After(completedHistory.CreatedAt) {
				progressDate = completedHistory.CreatedAt.Add(-time.Hour * 24) // 1 hari sebelum completed
			}
		}

		// Pilih deskripsi progress secara acak
		progressDesc := progressDescriptions[rand.Intn(len(progressDescriptions))]

		// Buat progress entry
		progress := models.WorkOrderProgress{
			WorkOrderID:      workOrder.ID,
			ProgressDesc:     progressDesc,
			ProgressQuantity: rand.Intn(workOrder.Quantity + 1),    // 0 sampai quantity
			CreatedAt:        progressDate,
			UpdatedAt:        progressDate,
		}

		database.DB.Create(&progress)
	}
}

// Inisialisasi random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}
