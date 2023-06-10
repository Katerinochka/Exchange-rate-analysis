package main

import (
	"encoding/xml"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Структура для парсинга xml
type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Date    string   `xml:"Date,attr"`
	Name    string   `xml:"name,attr"`
	Valutes []Valute `xml:"Valute"`
}

// Структура для парсинга xml
type Valute struct {
	XMLName  xml.Name `xml:"Valute"`
	ID       string   `xml:"ID,attr"`
	NumCode  int      `xml:"NumCode,omitempty"`
	CharCode string   `xml:"CharCode,omitempty"`
	Nominal  int      `xml:"Nominal,omitempty"`
	Name     string   `xml:"Name,omitempty"`
	Value    string   `xml:"Value,omitempty"`
}

// Структура для ответа
type AnswerValute struct {
	CharCode          string
	MaxValue          float64
	MaxDate           string
	MinValue          float64
	MinDate           string
	SumAverageRuble   float64
	CountAverageRuble int
}

// Делаем запрос по url с нужной датой
func RequestByDate(date string) (*http.Request, error) {
	req, err := http.NewRequest("GET", "http://www.cbr.ru/scripts/XML_daily.asp?date_req="+date, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// Получаем ответ от сервера
func ResponceBody(req *http.Request) (io.ReadCloser, error) {
	req.Header.Set("User-Agent", "MyUserAgent/1.0") // Добавим немного инфы о себе для сервера
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// Меняем кодировку ответа от сервера
func DecoderToUTF8(respBody io.ReadCloser) ([]byte, error) {
	decoder := charmap.Windows1252.NewDecoder()
	reader := decoder.Reader(respBody)
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return b[45:], nil
}

// Парсим xml
func ParseXML(respXML []byte, data *ValCurs) error {
	err := xml.Unmarshal(respXML, data)
	return err
}

// Общая функция получения данных с сервера
// Возвращает распарсенный xml в виде структуры
func GetData(date string) ValCurs {
	req, err := RequestByDate(date)
	if err != nil {
		err = fmt.Errorf("The request failed:  %s", err.Error())
		fmt.Println(err.Error())
	}

	respBody, err := ResponceBody(req)
	if err != nil {
		err = fmt.Errorf("The responce failed:  %s", err.Error())
		fmt.Println(err.Error())
	}
	defer respBody.Close()

	respXML, err := DecoderToUTF8(respBody)
	if err != nil {
		err = fmt.Errorf("Decoding error:  %s", err.Error())
		fmt.Println(err.Error())
	}

	var data ValCurs
	err = ParseXML(respXML, &data)
	if err != nil {
		err = fmt.Errorf("Parsing error:  %s", err.Error())
		fmt.Println(err.Error())
	}
	return data
}

// Заполнение структуры ответа программы после первого запроса с сервера
func AddFirstDataToAnswer(answer *[]AnswerValute, data *ValCurs) {
	oneAns := AnswerValute{}
	for _, valute := range data.Valutes {
		oneAns.CharCode = valute.CharCode
		curValue, _ := strconv.ParseFloat(strings.Replace(valute.Value, ",", ".", -1), 64) // в xml это поле написано через ",", поэтому сразу во float парсить не получается
		oneAns.MinValue = curValue / float64(valute.Nominal)                               // приводим все значения валют к одинаковому номиналу - единице
		oneAns.MinDate = data.Date
		oneAns.MaxValue = curValue / float64(valute.Nominal)
		oneAns.MaxDate = data.Date
		*answer = append(*answer, oneAns)
	}
}

// Вычисление min, max, dates, avg
func CalculateMaxMinAvg(answer []AnswerValute, data *ValCurs) {
	for i, valute := range data.Valutes {
		curValue, _ := strconv.ParseFloat(strings.Replace(valute.Value, ",", ".", -1), 64)
		if curValue/float64(valute.Nominal) > answer[i].MaxValue {
			answer[i].MaxValue = curValue / float64(valute.Nominal)
			answer[i].MaxDate = data.Date
		} else if curValue/float64(valute.Nominal) < answer[i].MinValue {
			answer[i].MinValue = curValue / float64(valute.Nominal)
			answer[i].MinDate = data.Date
		}
		answer[i].SumAverageRuble += float64(valute.Nominal) / curValue
		answer[i].CountAverageRuble++
	}
}

// Печать результата программы
func PrintAnswer(answer []AnswerValute) {
	fmt.Println("\t|\tMAX\t|   DateMAX\t|\tMIN\t|   DateMIN\t|\tAVG Ruble")
	for _, valute := range answer {
		fmt.Printf(" %s\t    %8.4f\t   %s\t   %8.4f\t   %s\t\t%8.4f\n", valute.CharCode, valute.MaxValue, valute.MaxDate, valute.MinValue, valute.MinDate, valute.SumAverageRuble/float64(valute.CountAverageRuble))
	}
}

func main() {
	countDays := 90
	timeNow := time.Now()             // Запоминаем сегодняшний день
	answer := make([]AnswerValute, 0) // Создаем пустой слайс ответов
	data := GetData(timeNow.Format("02/01/2006"))
	AddFirstDataToAnswer(&answer, &data) // Заполняем слайc ответов первым респонсом сервера
	for i := 1; i < countDays; i++ {     // С текущего дня по дню назад запрашиваем данные и делаем нужные вычисления
		data = GetData(timeNow.AddDate(0, 0, -i).Format("02/01/2006"))
		CalculateMaxMinAvg(answer, &data)
	}
	PrintAnswer(answer) // печатаем ответ
}
