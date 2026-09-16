package recognize

import (
	"fmt"
	"strconv"
	"strings"
)

// placeholderYear — год, который модель ставит при отсутствии года на картинке; високосный, чтобы 29 февраля разбиралось.
const placeholderYear = 2000

// System — системный промпт: форма JSON диктуется текстом, потому что схему в format облачная модель игнорирует.
func System(input Input) string {
	var b strings.Builder

	fmt.Fprintf(&b, `Ты разбираешь скриншоты банковских приложений: уведомления об операциях и списки операций.
Картинок: %d. Каждая картинка приходит отдельным сообщением с подписью «Картинка K», K от 1 до %d.

Ответь одним JSON-объектом ровно такой формы, без пояснений:
{"items":[{"source":1,"amount":"1234.50","currency":"₽","type":"expense","date":"2000-09-14",`+
		`"date_text":"14 сентября","year_present":false,"description":"Пятёрочка","category":"Еда / Продукты"}],`+
		`"incomplete":false}

Правила:
- items — по элементу на каждую покупку, оплату, списание или зачисление; нет операций — пустой массив.
- source — номер K картинки, на которой видна операция.
- amount — сумма строкой: только цифры и десятичная точка, без знака, пробелов и валюты.
- currency — валюта, как она указана на картинке (₽, руб., $, EUR), или null, если не указана.
- type — "expense" для покупок и списаний, "income" для зачислений.
- date — дата операции YYYY-MM-DD. Если год рядом с операцией не виден, поставь год %d и year_present=false.
  Если вместо даты написано «сегодня» или «вчера», или даты нет — null.
- date_text — дата так, как она написана на картинке, или null.
- year_present — true, только если год явно виден.
- description — продавец или назначение платежа, как на картинке.
- category — категория из списка ниже, строкой после двоеточия, только того же type и только если подходит
  однозначно; иначе null.
- incomplete — true, если список операций обрезан и часть операций не видна.
- Не включай: остатки и балансы, кредитные лимиты, комиссии отдельной строкой, итоги за период,
  переводы между своими счетами.
`, len(input.Images), len(input.Images), placeholderYear)

	b.WriteString("\nКатегории:\n")

	if len(input.Categories) == 0 {
		b.WriteString("(нет — category всегда null)\n")
	}

	for _, c := range input.Categories {
		fmt.Fprintf(&b, "%s: %s\n", c.Type, c.Path)
	}

	return b.String()
}

// UserText — подпись сообщения с картинкой i (индекс в Input.Images).
func UserText(i int) string {
	return "Картинка " + strconv.Itoa(i+1)
}
