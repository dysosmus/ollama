package lambda

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jmorganca/ollama/server"
)

func main() {
	lambda.Start(server.LambdaGenerateHandler)
}
