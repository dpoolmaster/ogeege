package render

import "strings"

// DefaultBrand — название продукта, если BRAND_NAME не задан.
const DefaultBrand = "Верно"

// Chrome содержит общие данные каркаса страницы.
type Chrome struct {
	Brand string
}

// BrandName возвращает название продукта без пустых значений.
func (c Chrome) BrandName() string {
	brand := strings.TrimSpace(c.Brand)
	if brand == "" {
		return DefaultBrand
	}

	return brand
}

// DocumentTitle собирает заголовок вкладки из названия страницы и бренда.
func DocumentTitle(title, brand string) string {
	title = strings.TrimSpace(title)
	brand = strings.TrimSpace(brand)
	switch {
	case title == "":
		return brand
	case brand == "" || title == brand:
		return title
	default:
		return title + " — " + brand
	}
}
