# Accept a payment

Securely accept payments online.

Build a payment form or use a prebuilt checkout page to start accepting online payments.

# Stripe-hosted page

> This is a Stripe-hosted page for when platform is web and ui is stripe-hosted. View the original doc at https://docs.stripe.com/payments/accept-a-payment?platform=web&ui=stripe-hosted.

Redirect to a Stripe-hosted payment page using [Stripe Checkout](https://docs.stripe.com/payments/checkout.md). See how this integration [compares to Stripe’s other integration types](https://docs.stripe.com/payments/online-payments.md#compare-features-and-availability).

Redirect to Stripe-hosted payment page

- 20 preset fonts
- 3 preset border radius
- Custom background and border color
- Custom logo

Try it out

## Redirect your customer to Stripe Checkout

Add a checkout button to your website that calls a server-side endpoint to create a [Checkout Session](https://docs.stripe.com/api/checkout/sessions/create.md).

You can also create a Checkout Session for an [existing customer](https://docs.stripe.com/payments/existing-customers.md?platform=web&ui=stripe-hosted), allowing you to prefill Checkout fields with known contact information and unify your purchase history for that customer.

```html
<html>
  <head>
    <title>Buy cool new product</title>
  </head>
  <body>
    <!-- Use action="/create-checkout-session.php" if your server is PHP based. -->
    <form action="/create-checkout-session" method="POST">
      <button type="submit">Checkout</button>
    </form>
  </body>
</html>
```

A Checkout Session is the programmatic representation of what your customer sees when they’re redirected to the payment form. You can configure it with options such as:

* [Line items](https://docs.stripe.com/api/checkout/sessions/create.md#create_checkout_session-line_items) to charge
* Currencies to use

You must populate `success_url` with the URL value of a page on your website that Checkout returns your customer to after they complete the payment. You can optionally also provide a `cancel_url` value of a page on your website that Checkout returns your customer to if they terminate the payment process before completion.

Checkout Sessions expire 24 hours after creation by default.

After creating a Checkout Session, redirect your customer to the [URL](https://docs.stripe.com/api/checkout/sessions/object.md#checkout_session_object-url) returned in the response.

```ruby
\# This example sets up an endpoint using the Sinatra framework.


require 'json'
require 'sinatra'
require 'stripe'
<<setup key>>

post '/create-checkout-session' do
  session = Stripe::Checkout::Session.create({
    line_items: [{
      price_data: {
        currency: 'usd',
        product_data: {
          name: 'T-shirt',
        },
        unit_amount: 2000,
      },
      quantity: 1,
    }],
    mode: 'payment',
    # These placeholder URLs will be replaced in a following step.
    success_url: 'https://example.com/success',
    cancel_url: 'https://example.com/cancel',
  })

  redirect session.url, 303
end
```

```python
\# This example sets up an endpoint using the Flask framework.
# Watch this video to get started: https://youtu.be/7Ul1vfmsDck.

import os
import stripe

from flask import Flask, redirect

app = Flask(__name__)

stripe.api_key = '<<secret key>>'

@app.route('/create-checkout-session', methods=['POST'])
def create_checkout_session():
  session = stripe.checkout.Session.create(
    line_items=[{
      'price_data': {
        'currency': 'usd',
        'product_data': {
          'name': 'T-shirt',
        },
        'unit_amount': 2000,
      },
      'quantity': 1,
    }],
    mode='payment',
    success_url='http://localhost:4242/success',
    cancel_url='http://localhost:4242/cancel',
  )

  return redirect(session.url, code=303)

if __name__== '__main__':
    app.run(port=4242)
```

```php
<?php

require 'vendor/autoload.php';

$stripe = new \Stripe\StripeClient('<<secret key>>');

$checkout_session = $stripe->checkout->sessions->create([
  'line_items' => [[
    'price_data' => [
      'currency' => 'usd',
      'product_data' => [
        'name' => 'T-shirt',
      ],
      'unit_amount' => 2000,
    ],
    'quantity' => 1,
  ]],
  'mode' => 'payment',
  'success_url' => 'http://localhost:4242/success',
  'cancel_url' => 'http://localhost:4242/cancel',
]);

header("HTTP/1.1 303 See Other");
header("Location: " . $checkout_session->url);
?>
```

```java
import java.util.HashMap;
import java.util.Map;
import static spark.Spark.get;
import static spark.Spark.post;
import static spark.Spark.port;
import static spark.Spark.staticFiles;

import com.stripe.Stripe;
import com.stripe.model.checkout.Session;
import com.stripe.param.checkout.SessionCreateParams;

public class Server {

  public static void main(String[] args) {
    port(4242);
    Stripe.apiKey = "<<secret key>>";

    post("/create-checkout-session", (request, response) -> {

      SessionCreateParams params =
        SessionCreateParams.builder()
          .setMode(SessionCreateParams.Mode.PAYMENT)
          .setSuccessUrl("http://localhost:4242/success")
          .setCancelUrl("http://localhost:4242/cancel")
          .addLineItem(
          SessionCreateParams.LineItem.builder()
            .setQuantity(1L)
            .setPriceData(
              SessionCreateParams.LineItem.PriceData.builder()
                .setCurrency("usd")
                .setUnitAmount(2000L)
                .setProductData(
                  SessionCreateParams.LineItem.PriceData.ProductData.builder()
                    .setName("T-shirt")
                    .build())
                .build())
            .build())
          .build();

      Session session = Session.create(params);

      response.redirect(session.getUrl(), 303);
      return "";
    });
  }
}
```

```javascript
// This example sets up an endpoint using the Express framework.

const express = require('express');
const app = express();
const stripe = require('stripe')('<<secret key>>')

app.post('/create-checkout-session', async (req, res) => {
  const session = await stripe.checkout.sessions.create({
    line_items: [
      {
        price_data: {
          currency: 'usd',
          product_data: {
            name: 'T-shirt',
          },
          unit_amount: 2000,
        },
        quantity: 1,
      },
    ],
    mode: 'payment',
    success_url: 'http://localhost:4242/success',
    cancel_url: 'http://localhost:4242/cancel',
  });

  res.redirect(303, session.url);
});

app.listen(4242, () => console.log(`Listening on port ${4242}!`));
```

```go
package main

import (
  "net/http"

  "github.com/labstack/echo"
  "github.com/labstack/echo/middleware"
  "github.com/stripe/stripe-go/v{{golang.major_version}}"
  "github.com/stripe/stripe-go/v{{golang.major_version}}/checkout/session"
)

// This example sets up an endpoint using the Echo framework.
// Watch this video to get started: https://youtu.be/ePmEVBu8w6Y.

func main() {
  stripe.Key = "<<secret key>>"

  e := echo.New()
  e.Use(middleware.Logger())
  e.Use(middleware.Recover())

  e.POST("/create-checkout-session", createCheckoutSession)

  e.Logger.Fatal(e.Start("localhost:4242"))
}

func createCheckoutSession(c echo.Context) (err error) {
  params := &stripe.CheckoutSessionParams{
    Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
    LineItems: []*stripe.CheckoutSessionLineItemParams{
      &stripe.CheckoutSessionLineItemParams{
        PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
          Currency: stripe.String("usd"),
          ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
            Name: stripe.String("T-shirt"),
          },
          UnitAmount: stripe.Int64(2000),
        },
        Quantity: stripe.Int64(1),
      },
    },
    SuccessURL: stripe.String("http://localhost:4242/success"),
    CancelURL:  stripe.String("http://localhost:4242/cancel"),
  }

  s, _ := session.New(params)

  if err != nil {
    return err
  }

  return c.Redirect(http.StatusSeeOther, s.URL)
}
```

```dotnet
// This example sets up an endpoint using the ASP.NET MVC framework.
// Watch this video to get started: https://youtu.be/2-mMOB8MhmE.

using System.Collections.Generic;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Options;
using Stripe;
using Stripe.Checkout;

namespace server.Controllers
{
  public class PaymentsController : Controller
  {
    public PaymentsController()
    {
      StripeConfiguration.ApiKey = "<<secret key>>";
    }

    [HttpPost("create-checkout-session")]
    public ActionResult CreateCheckoutSession()
    {
      var options = new SessionCreateOptions
      {
        LineItems = new List<SessionLineItemOptions>
        {
          new SessionLineItemOptions
          {
            PriceData = new SessionLineItemPriceDataOptions
            {
              UnitAmount = 2000,
              Currency = "usd",
              ProductData = new SessionLineItemPriceDataProductDataOptions
              {
                Name = "T-shirt",
              },
            },
            Quantity = 1,
          },
        },
        Mode = "payment",
        SuccessUrl = "http://localhost:4242/success",
        CancelUrl = "http://localhost:4242/cancel",
      };

      var service = new SessionService();
      Session session = service.Create(options);

      Response.Headers.Add("Location", session.Url);
      return new StatusCodeResult(303);
    }
  }
}
```

### Payment methods

By default, Stripe enables cards and other common payment methods. You can turn individual payment methods on or off in the [Stripe Dashboard](https://dashboard.stripe.com/settings/payment_methods). In Checkout, Stripe evaluates the currency and any restrictions, then dynamically presents the supported payment methods to the customer.

To see how your payment methods appear to customers, enter a transaction ID or set an order amount and currency in the Dashboard.

You can enable Apple Pay and Google Pay in your [payment methods settings](https://dashboard.stripe.com/settings/payment_methods). By default, Apple Pay is enabled and Google Pay is disabled. However, in some cases Stripe filters them out even when they’re enabled. We filter Google Pay if you [enable automatic tax](https://docs.stripe.com/tax/checkout.md) without collecting a shipping address.

Checkout’s Stripe-hosted pages don’t need integration changes to enable Apple Pay or Google Pay. Stripe handles these payments the same way as other card payments.

### Confirm your endpoint

Confirm your endpoint is accessible by starting your web server (for example, `localhost:4242`) and running the following command:

```bash
curl -X POST -is "http://localhost:4242/create-checkout-session" -d ""
```

You should see a response in your terminal that looks like this:

```bash
HTTP/1.1 303 See Other
Location: https://checkout.stripe.com/c/pay/cs_test_...
...
```

### Testing

You should now have a working checkout button that redirects your customer to Stripe Checkout.

1. Click the checkout button.
1. You’re redirected to the Stripe Checkout payment form.

If your integration isn’t working:

1. Open the Network tab in your browser’s developer tools.
1. Click the checkout button and confirm it sent an XHR request to your server-side endpoint (`POST /create-checkout-session`).
1. Verify the request is returning a 200 status.
1. Use `console.log(session)` inside your button click listener to confirm the correct data returned.

## Show a success page

It’s important for your customer to see a success page after they successfully submit the payment form. Host this success page on your site.

Create a minimal success page:

```html
<html>
  <head><title>Thanks for your order!</title></head>
  <body>
    <h1>Thanks for your order!</h1>
    <p>
      We appreciate your business!
      If you have any questions, please email
      <a href="mailto:orders@example.com">orders@example.com</a>.
    </p>
  </body>
</html>
```

Next, update the Checkout Session creation endpoint to use this new page:

```dotnet
StripeConfiguration.ApiKey = "<<secret key>>";

var options = new Stripe.Checkout.SessionCreateOptions
{
    LineItems = new List<Stripe.Checkout.SessionLineItemOptions>
    {
        new Stripe.Checkout.SessionLineItemOptions
        {
            PriceData = new Stripe.Checkout.SessionLineItemPriceDataOptions
            {
                Currency = "usd",
                ProductData = new Stripe.Checkout.SessionLineItemPriceDataProductDataOptions
                {
                    Name = "T-shirt",
                },
                UnitAmount = 2000,
            },
            Quantity = 1,
        },
    },
    Mode = "payment",
    SuccessUrl = "http://localhost:4242/success.html",
    CancelUrl = "http://localhost:4242/cancel.html",
};
var service = new Stripe.Checkout.SessionService();
Stripe.Checkout.Session session = service.Create(options);
```

```go
stripe.Key = "<<secret key>>"

params := &stripe.CheckoutSessionParams{
  LineItems: []*stripe.CheckoutSessionLineItemParams{
    &stripe.CheckoutSessionLineItemParams{
      PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
        Currency: stripe.String(string(stripe.CurrencyUSD)),
        ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
          Name: stripe.String("T-shirt"),
        },
        UnitAmount: stripe.Int64(2000),
      },
      Quantity: stripe.Int64(1),
    },
  },
  Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
  SuccessURL: stripe.String("http://localhost:4242/success.html"),
  CancelURL: stripe.String("http://localhost:4242/cancel.html"),
};
result, err := session.New(params);
```

```java
Stripe.apiKey = "<<secret key>>";

SessionCreateParams params =
  SessionCreateParams.builder()
    .addLineItem(
      SessionCreateParams.LineItem.builder()
        .setPriceData(
          SessionCreateParams.LineItem.PriceData.builder()
            .setCurrency("usd")
            .setProductData(
              SessionCreateParams.LineItem.PriceData.ProductData.builder()
                .setName("T-shirt")
                .build()
            )
            .setUnitAmount(2000L)
            .build()
        )
        .setQuantity(1L)
        .build()
    )
    .setMode(SessionCreateParams.Mode.PAYMENT)
    .setSuccessUrl("http://localhost:4242/success.html")
    .setCancelUrl("http://localhost:4242/cancel.html")
    .build();

Session session = Session.create(params);
```

```node
const stripe = require('stripe')('<<secret key>>');

const session = await stripe.checkout.sessions.create({
  line_items: [
    {
      price_data: {
        currency: 'usd',
        product_data: {
          name: 'T-shirt',
        },
        unit_amount: 2000,
      },
      quantity: 1,
    },
  ],
  mode: 'payment',
  success_url: 'http://localhost:4242/success.html',
  cancel_url: 'http://localhost:4242/cancel.html',
});
```

```python
import stripe
stripe.api_key = "<<secret key>>"

session = stripe.checkout.Session.create(
  line_items=[
    {
      "price_data": {"currency": "usd", "product_data": {"name": "T-shirt"}, "unit_amount": 2000},
      "quantity": 1,
    },
  ],
  mode="payment",
  success_url="http://localhost:4242/success.html",
  cancel_url="http://localhost:4242/cancel.html",
)
```

```php
$stripe = new \Stripe\StripeClient('<<secret key>>');

$session = $stripe->checkout->sessions->create([
  'line_items' => [
    [
      'price_data' => [
        'currency' => 'usd',
        'product_data' => ['name' => 'T-shirt'],
        'unit_amount' => 2000,
      ],
      'quantity' => 1,
    ],
  ],
  'mode' => 'payment',
  'success_url' => 'http://localhost:4242/success.html',
  'cancel_url' => 'http://localhost:4242/cancel.html',
]);
```

```ruby
Stripe.api_key = '<<secret key>>'

session = Stripe::Checkout::Session.create({
  line_items: [
    {
      price_data: {
        currency: 'usd',
        product_data: {name: 'T-shirt'},
        unit_amount: 2000,
      },
      quantity: 1,
    },
  ],
  mode: 'payment',
  success_url: 'http://localhost:4242/success.html',
  cancel_url: 'http://localhost:4242/cancel.html',
})
```

If you want to customize your success page, read the [custom success page](https://docs.stripe.com/payments/checkout/custom-success-page.md) guide.

### Testing

1. Click your checkout button.
1. Fill out the payment details with the test card information:
   - Enter `4242 4242 4242 4242` as the card number.
   - Enter any future date for card expiry.
   - Enter any 3-digit number for CVC.
   - Enter any billing postal code.
1. Click **Pay**.
1. You’re redirected to your new success page.

Next, find the new payment in the Stripe Dashboard. Successful payments appear in the Dashboard’s [list of payments](https://dashboard.stripe.com/payments). When you click a payment, it takes you to the payment details page. The **Checkout summary** section contains billing information and the list of items purchased, which you can use to manually fulfill the order.

## Handle post-payment events

Stripe sends a [checkout.session.completed](https://docs.stripe.com/api/events/types.md#event_types-checkout.session.completed) event when a customer completes a Checkout Session payment. Use the [Dashboard webhook tool](https://dashboard.stripe.com/webhooks) or follow the [webhook guide](https://docs.stripe.com/webhooks/quickstart.md) to receive and handle these events, which might trigger you to:

* Send an order confirmation email to your customer.
* Log the sale in a database.
* Start a shipping workflow.

Listen for these events rather than waiting for your customer to be redirected back to your website. Triggering fulfillment only from your Checkout landing page is unreliable. Setting up your integration to listen for asynchronous events allows you to accept [different types of payment methods](https://stripe.com/payments/payment-methods-guide) with a single integration.

Learn more in our [fulfillment guide for Checkout](https://docs.stripe.com/checkout/fulfillment.md).

Handle the following events when collecting payments with the Checkout:

| Event                                                                                                                                        | Description                                                                                | Action                                                                                      |
| -------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------- |
| [checkout.session.completed](https://docs.stripe.com/api/events/types.md#event_types-checkout.session.completed)                             | Sent when a customer successfully completes a Checkout Session.                            | Send the customer an order confirmation and *fulfill* their order.                          |
| [checkout.session.async_payment_succeeded](https://docs.stripe.com/api/events/types.md#event_types-checkout.session.async_payment_succeeded) | Sent when a payment made with a delayed payment method, such as ACH direct debt, succeeds. | Send the customer an order confirmation and *fulfill* their order.                          |
| [checkout.session.async_payment_failed](https://docs.stripe.com/api/events/types.md#event_types-checkout.session.async_payment_failed)       | Sent when a payment made with a delayed payment method, such as ACH direct debt, fails.    | Notify the customer of the failure and bring them back on-session to attempt payment again. |

## Test your integration

To test your Stripe-hosted payment form integration:

1. Create a Checkout Session.
1. Fill out the payment details with a method from the following table.
   - Enter any future date for card expiry.
   - Enter any 3-digit number for CVC.
   - Enter any billing postal code.
1. Click **Pay**. You’re redirected to your `success_url`.
1. Go to the Dashboard and look for the payment on the [Transactions page](https://dashboard.stripe.com/test/payments?status%5B0%5D=successful). If your payment succeeded, you’ll see it in that list.
1. Click your payment to see more details, like a Checkout summary with billing information and the list of purchased items. You can use this information to fulfill the order.

Learn more about [testing your integration](https://docs.stripe.com/testing.md).

| Card number         | Scenario                                                            | How to test                                                                                           |
| ------------------- | ------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| 4242424242424242    | The card payment succeeds and doesn’t require authentication.       | Fill out the credit card form using the credit card number with any expiration, CVC, and postal code. |
| 4000002500003155    | The card payment requires *authentication*.                         | Fill out the credit card form using the credit card number with any expiration, CVC, and postal code. |
| 4000000000009995    | The card is declined with a decline code like `insufficient_funds`. | Fill out the credit card form using the credit card number with any expiration, CVC, and postal code. |
| 6205500000000000004 | The UnionPay card has a variable length of 13-19 digits.            | Fill out the credit card form using the credit card number with any expiration, CVC, and postal code. |

| Payment method | Scenario                                                                                                                                                                   | How to test                                                                                                                                              |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
|                | Your customer fails to authenticate on the redirect page for a redirect-based and immediate notification payment method.                                                   | Choose any redirect-based payment method, fill out the required details, and confirm the payment. Then click **Fail test payment** on the redirect page. |
| Pay by Bank    | Your customer successfully pays with a redirect-based and [delayed notification](https://docs.stripe.com/payments/payment-methods.md#payment-notification) payment method. | Choose the payment method, fill out the required details, and confirm the payment. Then click **Complete test payment** on the redirect page.            |
| Pay by Bank    | Your customer fails to authenticate on the redirect page for a redirect-based and delayed notification payment method.                                                     | Choose the payment method, fill out the required details, and confirm the payment. Then click **Fail test payment** on the redirect page.                |

| Payment method    | Scenario                                                                                          | How to test                                                                                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| SEPA Direct Debit | Your customer successfully pays with SEPA Direct Debit.                                           | Fill out the form using the account number `AT321904300235473204`. The confirmed PaymentIntent initially transitions to processing, then transitions to the succeeded status three minutes later. |
| SEPA Direct Debit | Your customer’s payment intent status transitions from `processing` to `requires_payment_method`. | Fill out the form using the account number `AT861904300235473202`.                                                                                                                                |

See [Testing](https://docs.stripe.com/testing.md) for additional information to test your integration.

### Test cards

| Number              | Description                                                   |
| ------------------- | ------------------------------------------------------------- |
| 4242 4242 4242 4242 | Succeeds and immediately processes the payment.               |
| 4000 0000 0000 3220 | Requires 3D Secure 2 authentication for a successful payment. |
| 4000 0000 0000 9995 | Always fails with a decline code of `insufficient_funds`.     |

## Create products and prices

You can [set up your Checkout Session](https://docs.stripe.com/payments/checkout/pay-what-you-want.md) to accept tips and donations, or sell pay-what-you-want products and services.

Before you create a Checkout Session, you can create *Products* and *Prices* upfront. Use products to represent different physical goods or levels of service, and *Prices* to represent each product’s pricing.

For example, you can create a T-shirt as a product with a price of 20 USD. This allows you to update and add prices without needing to change the details of your underlying products. You can either create products and prices with the Stripe Dashboard or API. Learn more about [how products and prices work](https://docs.stripe.com/products-prices/how-products-and-prices-work.md).

The API only requires a `name` to create a [Product](https://docs.stripe.com/api/products.md). Checkout displays the product `name`, `description`, and `images` that you supply.

```dotnet
StripeConfiguration.ApiKey = "<<secret key>>";

var options = new ProductCreateOptions { Name = "T-shirt" };
var service = new ProductService();
Product product = service.Create(options);
```

```go
stripe.Key = "<<secret key>>"

params := &stripe.ProductParams{Name: stripe.String("T-shirt")};
result, err := product.New(params);
```

```java
Stripe.apiKey = "<<secret key>>";

ProductCreateParams params = ProductCreateParams.builder().setName("T-shirt").build();

Product product = Product.create(params);
```

```node
const stripe = require('stripe')('<<secret key>>');

const product = await stripe.products.create({
  name: 'T-shirt',
});
```

```python
import stripe
stripe.api_key = "<<secret key>>"

product = stripe.Product.create(name="T-shirt")
```

```php
$stripe = new \Stripe\StripeClient('<<secret key>>');

$product = $stripe->products->create(['name' => 'T-shirt']);
```

```ruby
Stripe.api_key = '<<secret key>>'

product = Stripe::Product.create({name: 'T-shirt'})
```

Next, create a [Price](https://docs.stripe.com/api/prices.md) to define how much to charge for your product. This includes how much the product costs and what currency to use.

```dotnet
StripeConfiguration.ApiKey = "<<secret key>>";

var options = new PriceCreateOptions
{
    Product = "<<product>>",
    UnitAmount = 2000,
    Currency = "usd",
};
var service = new PriceService();
Price price = service.Create(options);
```

```go
stripe.Key = "<<secret key>>"

params := &stripe.PriceParams{
  Product: stripe.String("<<product>>"),
  UnitAmount: stripe.Int64(2000),
  Currency: stripe.String(string(stripe.CurrencyUSD)),
};
result, err := price.New(params);
```

```java
Stripe.apiKey = "<<secret key>>";

PriceCreateParams params =
  PriceCreateParams.builder()
    .setProduct("<<product>>")
    .setUnitAmount(2000L)
    .setCurrency("usd")
    .build();

Price price = Price.create(params);
```

```node
const stripe = require('stripe')('<<secret key>>');

const price = await stripe.prices.create({
  product: '<<product>>',
  unit_amount: 2000,
  currency: 'usd',
});
```

```python
import stripe
stripe.api_key = "<<secret key>>"

price = stripe.Price.create(
  product="<<product>>",
  unit_amount=2000,
  currency="usd",
)
```

```php
$stripe = new \Stripe\StripeClient('<<secret key>>');

$price = $stripe->prices->create([
  'product' => '<<product>>',
  'unit_amount' => 2000,
  'currency' => 'usd',
]);
```

```ruby
Stripe.api_key = '<<secret key>>'

price = Stripe::Price.create({
  product: '<<product>>',
  unit_amount: 2000,
  currency: 'usd',
})
```

Copy products created in a sandbox to live mode so that you don’t need to re-create them. In the Product detail view in the Dashboard, click **Copy to live mode** in the upper right corner. You can only do this once for each product created in a sandbox. Subsequent updates to the test product aren’t reflected for the live product.

Make sure you’re in a sandbox by clicking **Sandboxes** within the Dashboard account picker. Next, define the items you want to sell. To create a new product and price:

- Navigate to the [Products](https://dashboard.stripe.com/test/products) section in the Dashboard.
- Click **Add product**.
- Select **One time** when setting the price.

Checkout displays the product name, description, and images that you supply.

Each price you create has an ID. When you create a Checkout Session, reference the price ID and quantity. If you’re selling in multiple currencies, make your Price *multi-currency*. Checkout automatically [determines the customer’s local currency](https://docs.stripe.com/payments/currencies/localize-prices/manual-currency-prices.md) and presents that currency if the Price supports it.

```dotnet
StripeConfiguration.ApiKey = "<<secret key>>";

var options = new Stripe.Checkout.SessionCreateOptions
{
    Mode = "payment",
    LineItems = new List<Stripe.Checkout.SessionLineItemOptions>
    {
        new Stripe.Checkout.SessionLineItemOptions { Price = "{{PRICE_ID}}", Quantity = 1 },
    },
    SuccessUrl = "https://example.com/success?session_id={CHECKOUT_SESSION_ID}",
    CancelUrl = "https://example.com/cancel",
};
var service = new Stripe.Checkout.SessionService();
Stripe.Checkout.Session session = service.Create(options);
```

```go
stripe.Key = "<<secret key>>"

params := &stripe.CheckoutSessionParams{
  Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
  LineItems: []*stripe.CheckoutSessionLineItemParams{
    &stripe.CheckoutSessionLineItemParams{
      Price: stripe.String("{{PRICE_ID}}"),
      Quantity: stripe.Int64(1),
    },
  },
  SuccessURL: stripe.String("https://example.com/success?session_id={CHECKOUT_SESSION_ID}"),
  CancelURL: stripe.String("https://example.com/cancel"),
};
result, err := session.New(params);
```

```java
Stripe.apiKey = "<<secret key>>";

SessionCreateParams params =
  SessionCreateParams.builder()
    .setMode(SessionCreateParams.Mode.PAYMENT)
    .addLineItem(
      SessionCreateParams.LineItem.builder().setPrice("{{PRICE_ID}}").setQuantity(1L).build()
    )
    .setSuccessUrl("https://example.com/success?session_id={CHECKOUT_SESSION_ID}")
    .setCancelUrl("https://example.com/cancel")
    .build();

Session session = Session.create(params);
```

```node
const stripe = require('stripe')('<<secret key>>');

const session = await stripe.checkout.sessions.create({
  mode: 'payment',
  line_items: [
    {
      price: '{{PRICE_ID}}',
      quantity: 1,
    },
  ],
  success_url: 'https://example.com/success?session_id={CHECKOUT_SESSION_ID}',
  cancel_url:
