package subscription

import "encore.dev"

const (
	N8NINSTANCE_PRODUCT_ID = "n8n_instance_product_id"
)

type Product struct {
	Key  string
	Name string
}

type Products struct {
	products map[string]*Product
}

func InitProducts(env encore.EnvironmentType) *Products {

	productIDs := make(map[string]*Product)

	if env == encore.EnvProduction {
		productIDs[N8NINSTANCE_PRODUCT_ID] = &Product{
			Key:  N8NINSTANCE_PRODUCT_ID,
			Name: "n8n Instance",
		}
	} else {
		productIDs[N8NINSTANCE_PRODUCT_ID] = &Product{
			Key:  N8NINSTANCE_PRODUCT_ID,
			Name: "n8n Instance",
		}
	}

	return &Products{
		products: productIDs,
	}
}

func (p *Products) GetProduct(key string) *Product {
	return p.products[key]
}
