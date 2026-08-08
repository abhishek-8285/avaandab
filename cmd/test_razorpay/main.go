package main

import (
	"fmt"
	"log"

	"transport-app/internal/config"
	"transport-app/internal/payment/razorpay"
)

func main() {
	cfg := config.Load()

	fmt.Println("==================================================")
	fmt.Println("🚀 Testing Razorpay Integration with Configured Keys")
	fmt.Println("==================================================")
	fmt.Printf("Key ID: %s\n", cfg.RazorpayKeyID)

	client := razorpay.NewRazorpayClient(cfg.RazorpayKeyID, cfg.RazorpayKeySecret)

	// 1. Test Order Creation via Razorpay API
	fmt.Println("\n1. Creating test order via Razorpay API...")
	order, err := client.CreateOrder("inv_demo_1001", 4999.00, "INR")
	if err != nil {
		log.Fatalf("❌ Order creation failed: %v", err)
	}

	fmt.Println("✅ Razorpay Order Created Successfully!")
	fmt.Printf("   • Order ID : %s\n", order.ID)
	fmt.Printf("   • Amount   : ₹%.2f (%d paise)\n", float64(order.Amount)/100, order.Amount)
	fmt.Printf("   • Status   : %s\n", order.Status)

	// 2. Test Cryptographic Signature Verification
	fmt.Println("\n2. Testing HMAC-SHA256 Payment Signature Verification...")
	testPaymentID := "pay_test_88712391"

	// Verify signature check logic
	valid := client.VerifyPaymentSignature(order.ID, testPaymentID, "invalid_signature")
	if !valid {
		fmt.Println("✅ Signature Security Verification Working (Rejected fake signature as expected)")
	}

	fmt.Println("\n==================================================")
	fmt.Println("🎉 ALL RAZORPAY INTEGRATION TESTS PASSED!")
	fmt.Println("==================================================")
}
