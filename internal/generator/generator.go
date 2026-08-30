package generator

import (
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/MarmulevSemyon/go-api-performance-lab/internal/model"
)

func GenerateOrders(count int) map[string]model.Order {
	gofakeit.Seed(1)

	orders := make(map[string]model.Order, count)

	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("order-%d", i)
		track := fmt.Sprintf("TRACK-%d", i)

		itemsCount := gofakeit.Number(1, 5)
		items := make([]model.Item, 0, itemsCount)

		goodsTotal := 0

		for j := 1; j <= itemsCount; j++ {
			price := gofakeit.Number(300, 5000)
			sale := gofakeit.Number(0, 30)
			totalPrice := price - price*sale/100
			goodsTotal += totalPrice

			items = append(items, model.Item{
				OrderID:     id,
				ChrtID:      int64(100000 + i*10 + j),
				TrackNumber: track,
				Price:       price,
				RID:         gofakeit.UUID(),
				Name:        gofakeit.ProductName(),
				Sale:        sale,
				Size:        gofakeit.RandomString([]string{"S", "M", "L", "XL"}),
				TotalPrice:  totalPrice,
				NmID:        int64(200000 + i*10 + j),
				Brand:       gofakeit.Company(),
				Status:      202,
			})
		}

		deliveryCost := gofakeit.Number(100, 700)

		order := model.Order{
			OrderUID:          id,
			TrackNumber:       track,
			Entry:             "WBIL",
			Locale:            "ru",
			InternalSignature: "",
			CustomerID:        fmt.Sprintf("customer-%d", i),
			DeliveryService:   gofakeit.RandomString([]string{"meest", "cdek", "boxberry"}),
			ShardKey:          fmt.Sprintf("%d", gofakeit.Number(1, 9)),
			SmID:              gofakeit.Number(1, 100),
			DateCreated:       time.Now().Add(-time.Duration(gofakeit.Number(1, 1000)) * time.Hour),
			OofShard:          "1",

			Delivery: model.Delivery{
				OrderID: id,
				Name:    gofakeit.Name(),
				Phone:   gofakeit.Phone(),
				Zip:     gofakeit.Zip(),
				City:    gofakeit.City(),
				Address: gofakeit.Street(),
				Region:  gofakeit.State(),
				Email:   gofakeit.Email(),
			},

			Payment: model.Payment{
				OrderID:      id,
				Transaction:  gofakeit.UUID(),
				RequestID:    "",
				Currency:     "RUB",
				Provider:     "wbpay",
				Amount:       float64(goodsTotal + deliveryCost),
				PaymentDT:    time.Now().Unix(),
				Bank:         gofakeit.RandomString([]string{"sber", "alpha", "tinkoff", "vtb"}),
				DeliveryCost: deliveryCost,
				GoodsTotal:   goodsTotal,
				CustomFee:    0,
			},

			Items: items,
		}

		orders[id] = order
	}

	return orders
}
