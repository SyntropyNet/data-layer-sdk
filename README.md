# Syntropy Data Layer SDK

Welcome to the [Data Layer SDK](https://github.com/SyntropyNet/data-layer-sdk) documentation. This SDK enables seamless integration with our Data Layer solution, allowing you to harness the power of real-time data streams in your Golang applications.

[data-layer-sdk](https://github.com/SyntropyNet/data-layer-sdk) is a [Golang library](https://github.com/SyntropyNet/data-layer-sdk) designed specifically for the Syntropy Data Layer project. Powered by the NATS messaging system, [data-layer-sdk](https://github.com/SyntropyNet/data-layer-sdk) abstracts away most of the intricate details of interacting 
with the Data Layer Broker network and the Blockchain to offer seamless integration between your Golang applications and the Syntropy Data Layer platform.

# Features

- **Subscribe to Existing Data Streams**: Easily subscribe to pre-existing data streams within the Syntropy Data Layer. Receive real-time updates and harness the power of real-time data insights in your Golang applications.

- **Publish New Data Streams**: Create and publish your own data streams directly from your Golang applications. Share data with other participants in the Data Layer, facilitating collaboration and enabling the creation of innovative data-driven solutions.

- **Support for JSON Messages**: Leverage the flexibility and interoperability of JSON messages. [data-layer-sdk](https://github.com/SyntropyNet/data-layer-sdk) provides support for handling JSON data, making it easy to work with complex data structures and seamlessly integrate with other systems and platforms. Even more, the SDK allows adding your own type marshallers to support any wire data format.

- **Customizable Connection Options**: Tailor the connection options to suit your specific needs. Configure parameters such as connection timeouts, retry mechanisms, and authentication details to ensure a secure and reliable connection to the Syntropy Data Layer platform.

# Installation

To install the SDK for Data Layer, you can use Go modules or include it as a dependency in your project. Here's an example of how to add it to your project:

```shell
go get github.com/SyntropyNet/data-layer-sdk
```

Make sure to import the package in your code:

```go
import "github.com/SyntropyNet/data-layer-sdk"
```

## Getting started
In order to implement a simple publisher, you may want to embed `service.Service` into your publisher's struct.
Then in the constructor call `Service.Configure` that will use options to configure the publisher.
After that it is as simple as running `Start` on your publisher.

The minimal example is as follows:

```go
type Publisher struct {
	*service.Service
}

// Define custom payload messages
type MyMessage struct {
	RequestWas []byte `json:"request"`
}

func New(o ...options.Option) (*Publisher, error) {
	ret := &Publisher{
		&service.Service{},
	}

	err := ret.Base.Configure(o...)
	if err != nil {
		return nil, fmt.Errorf("failed configuring the publisher: %w", err)
	}

	return ret, nil
}

func (p *Publisher) Start() <-chan error {
	err := p.subscribe()
	if err != nil {
		go func() {
			p.ErrCh <- err
		}()
		return p.ErrCh
	}

	return p.Base.Start()
}

func (p *Publisher) subscribe() error {
  // Do any necessary subscriptions
	return nil
}
```

# Contributing

We welcome contributions from the community! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request on the [GitHub repository](https://github.com/SyntropyNet/data-layer-sdk). We appreciate your feedback and collaboration in making this SDK even better. 

## Contribution Guidelines

To contribute to this project, please follow the guidelines outlined in the [Contribute.md](CONTRIBUTE.md) file. It covers important information about how to submit bug reports, suggest new features, and submit pull requests.

## Code of Conduct
This project adheres to a [Code of Conduct](CODE-OF-CONDUCT.md) to ensure a welcoming and inclusive environment for all contributors. Please review the guidelines and make sure to follow them in all interactions within the project.

## Commit Message Format
When making changes to the codebase, it's important to follow a consistent commit message format. Please refer to the [Commit Message Format](commit-template.md) for details on how to structure your commit messages.

## Pull Request Template
To streamline the pull request process, we have provided a [Pull Request Template](pull-request-template.md) that includes the necessary sections for describing your changes, related issues, proposed changes, and any additional information. Make sure to fill out the template when submitting a pull request.

We appreciate your contributions and thank you for your support in making this project better!

# Support

If you encounter any difficulties or have questions regarding the Data Layer SDK, please reach out to our support team at [Discord #developer-discussion](https://discord.com/channels/503896258881126401/1125658694399561738). We are here to assist you and ensure a smooth experience with our SDK.

We hope this documentation provides you with a comprehensive understanding of the Golang SDK for the Data Layer. Happy coding with real-time data streams and enjoy the power of the Data Layer in your Golang applications!
