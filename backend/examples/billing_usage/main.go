package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/pkg/models"
)

func main() {
	fmt.Println("=== MBFlow Billing System Demo ===")

	userID := "user-" + uuid.New().String()[:8]

	demoAccountOperations(userID)
	demoResourceManagement(userID)
	demoTransactionFlow(userID)
	demoPricingPlans()
}

func demoAccountOperations(userID string) {
	fmt.Println("1. Account Operations Demo")
	fmt.Println("---------------------------")

	account := models.NewAccount(userID)
	fmt.Printf("Created account for user %s\n", userID)
	fmt.Printf("Initial balance: $%.2f %s\n", account.Balance, account.Currency)
	fmt.Printf("Account status: %s\n", account.Status)

	if err := account.Deposit(100.00); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("After deposit: $%.2f\n", account.Balance)

	if account.CanCharge(25.00) {
		if err := account.Charge(25.00); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("After charge: $%.2f\n", account.Balance)
	}

	fmt.Printf("Has sufficient balance for $100? %v\n", account.HasSufficientBalance(100.00))
	fmt.Printf("Can charge $50? %v\n", account.CanCharge(50.00))

	account.Suspend()
	fmt.Printf("After suspension - can charge? %v\n", account.CanCharge(10.00))

	account.Activate()
	fmt.Printf("After activation - can charge? %v\n\n", account.CanCharge(10.00))
}

func demoResourceManagement(userID string) {
	fmt.Println("2. Resource Management Demo")
	fmt.Println("----------------------------")

	storage := models.NewFileStorageResource(userID, "My Project Storage")
	fmt.Printf("Created file storage resource\n")
	fmt.Printf("Storage limit: %d bytes (%.2f MB)\n",
		storage.StorageLimitBytes,
		float64(storage.StorageLimitBytes)/(1024*1024))

	fileSizes := []int64{
		1024 * 1024,
		2 * 1024 * 1024,
		1536 * 1024,
	}

	for i, size := range fileSizes {
		sizeMB := float64(size) / (1024 * 1024)
		if storage.CanAddFile(size) {
			if err := storage.AddFile(size); err != nil {
				fmt.Printf("Error adding file %d: %v\n", i+1, err)
			} else {
				fmt.Printf("Added file %d (%.2f MB)\n", i+1, sizeMB)
				fmt.Printf("  Usage: %.1f%% (%d/%d bytes)\n",
					storage.GetUsagePercent(),
					storage.UsedStorageBytes,
					storage.StorageLimitBytes)
				fmt.Printf("  Files: %d\n", storage.FileCount)
				fmt.Printf("  Available: %.2f MB\n",
					float64(storage.GetAvailableSpace())/(1024*1024))
			}
		} else {
			fmt.Printf("Cannot add file %d (%.2f MB) - storage limit exceeded\n", i+1, sizeMB)
		}
	}

	storage.RemoveFile(fileSizes[0])
	fmt.Printf("\nAfter removing first file:\n")
	fmt.Printf("  Usage: %.1f%%\n", storage.GetUsagePercent())
	fmt.Printf("  Files: %d\n", storage.FileCount)

	if err := storage.UpdateLimit(10 * 1024 * 1024); err != nil {
		fmt.Printf("Error updating limit: %v\n", err)
	} else {
		fmt.Printf("  Updated limit to 10 MB\n")
		fmt.Printf("  Available: %.2f MB\n\n",
			float64(storage.GetAvailableSpace())/(1024*1024))
	}
}

func demoTransactionFlow(userID string) {
	fmt.Println("3. Transaction Flow Demo")
	fmt.Println("-------------------------")

	account := models.NewAccount(userID)
	account.Deposit(100.00)

	scenarios := []struct {
		txType      models.TransactionType
		amount      float64
		description string
	}{
		{models.TransactionTypeDeposit, 50.00, "Initial deposit"},
		{models.TransactionTypeCharge, 25.00, "Premium plan subscription"},
		{models.TransactionTypeCharge, 10.00, "Storage overage"},
		{models.TransactionTypeRefund, 5.00, "Partial refund"},
		{models.TransactionTypeCharge, 200.00, "Should fail - insufficient balance"},
	}

	for i, scenario := range scenarios {
		fmt.Printf("\nTransaction %d: %s ($%.2f)\n", i+1, scenario.txType, scenario.amount)

		tx := &models.Transaction{
			ID:             uuid.New().String(),
			AccountID:      account.ID,
			Type:           scenario.txType,
			Amount:         scenario.amount,
			Currency:       account.Currency,
			Status:         models.TransactionStatusPending,
			Description:    scenario.description,
			IdempotencyKey: uuid.New().String(),
			BalanceBefore:  account.Balance,
			CreatedAt:      time.Now(),
		}

		if err := tx.Validate(); err != nil {
			fmt.Printf("  Validation error: %v\n", err)
			continue
		}

		var err error
		switch scenario.txType {
		case models.TransactionTypeDeposit:
			err = account.Deposit(scenario.amount)
		case models.TransactionTypeCharge:
			err = account.Charge(scenario.amount)
		case models.TransactionTypeRefund:
			err = account.Deposit(scenario.amount)
		}

		if err != nil {
			tx.Fail()
			fmt.Printf("  Status: %s\n", tx.Status)
			fmt.Printf("  Error: %v\n", err)
			fmt.Printf("  Balance: $%.2f (unchanged)\n", account.Balance)
		} else {
			tx.BalanceAfter = account.Balance
			tx.Complete()
			fmt.Printf("  Status: %s\n", tx.Status)
			fmt.Printf("  Balance: $%.2f → $%.2f\n", tx.BalanceBefore, tx.BalanceAfter)
		}
	}
	fmt.Println()
}

func demoPricingPlans() {
	fmt.Println("4. Pricing Plans Demo")
	fmt.Println("----------------------")

	plans := []models.PricingPlan{
		{
			ID:                uuid.New().String(),
			ResourceType:      models.ResourceTypeFileStorage,
			Name:              "Free",
			Description:       "Free tier for personal projects",
			PricePerUnit:      0,
			Unit:              "month",
			StorageLimitBytes: intPtr(5 * 1024 * 1024),
			BillingPeriod:     models.BillingPeriodMonthly,
			PricingModel:      models.PricingModelFixed,
			IsFree:            true,
			IsActive:          true,
			CreatedAt:         time.Now(),
		},
		{
			ID:                uuid.New().String(),
			ResourceType:      models.ResourceTypeFileStorage,
			Name:              "Premium",
			Description:       "Premium plan for professional use",
			PricePerUnit:      9.99,
			Unit:              "month",
			StorageLimitBytes: intPtr(100 * 1024 * 1024),
			BillingPeriod:     models.BillingPeriodMonthly,
			PricingModel:      models.PricingModelFixed,
			IsFree:            false,
			IsActive:          true,
			CreatedAt:         time.Now(),
		},
		{
			ID:                uuid.New().String(),
			ResourceType:      models.ResourceTypeFileStorage,
			Name:              "Enterprise Annual",
			Description:       "Annual enterprise plan",
			PricePerUnit:      999.00,
			Unit:              "year",
			StorageLimitBytes: intPtr(1024 * 1024 * 1024),
			BillingPeriod:     models.BillingPeriodAnnual,
			PricingModel:      models.PricingModelFixed,
			IsFree:            false,
			IsActive:          true,
			CreatedAt:         time.Now(),
		},
	}

	for _, plan := range plans {
		if err := plan.Validate(); err != nil {
			fmt.Printf("Invalid plan: %v\n", err)
			continue
		}

		fmt.Printf("\nPlan: %s\n", plan.Name)
		fmt.Printf("  Description: %s\n", plan.Description)
		fmt.Printf("  Resource Type: %s\n", plan.ResourceType)
		fmt.Printf("  Pricing Model: %s\n", plan.PricingModel)

		if plan.IsFree {
			fmt.Printf("  Price: FREE\n")
		} else {
			fmt.Printf("  Monthly Price: $%.2f\n", plan.GetMonthlyPrice())
			fmt.Printf("  Annual Price: $%.2f\n", plan.GetAnnualPrice())
			if plan.BillingPeriod == models.BillingPeriodAnnual {
				savings := (plan.GetMonthlyPrice() * 12) - plan.GetAnnualPrice()
				if savings > 0 {
					fmt.Printf("  Annual Savings: $%.2f\n", savings)
				}
			}
		}

		if plan.StorageLimitBytes != nil {
			fmt.Printf("  Storage Limit: %.0f MB\n",
				float64(*plan.StorageLimitBytes)/(1024*1024))
		}
		fmt.Printf("  Available: %v\n", plan.IsAvailable())
	}
	fmt.Println()
}

func intPtr(i int64) *int64 {
	return &i
}
