---
icon: material/credit-card-outline
---

# 💸 Pay out to people

Add money to your Stripe balance and pay out to sellers or service providers.

- **Stripe compatibility:** *Connect* (Connect is Stripe's solution for multi-party businesses, such as marketplace or software platforms, to route payments between sellers, customers, and other recipients)
- **Requires:** Stripe account, configured Connect platform
- **Good for:** Marketplaces, platforms
- **Pricing:** [Connect pricing](https://stripe.com/connect/pricing)

Use this guide to learn how to add funds to your account balance and transfer the funds into your users’ bank accounts, without processing payments through Stripe. This guide uses an example of a Q&A product that pays its writers a portion of the advertising revenue that their answers generate. The platform and connected accounts are both in the US.

For businesses using automatic payouts, funds added to the payments balance in excess of the [minimum balance](https://docs.stripe.com/payouts/minimum-balances-for-automatic-payouts.md) are paid out in the next payout. You can configure your payout schedule and minimum balance settings in your [Payout settings](https://dashboard.stripe.com/settings/payouts).

> Only [team members](https://docs.stripe.com/get-started/account/teams.md) with administrator access to the platform Stripe account and [two-factor authentication](https://support.stripe.com/questions/how-do-i-enable-two-step-verification) enabled can add funds.

## 📌 Prerequisites

### Prerequisites

- [ ] [Register your platform](https://dashboard.stripe.com/connect)

- [ ] Add business details to [activate your account](https://dashboard.stripe.com/account/onboarding)

- [ ] [Complete your platform profile](https://dashboard.stripe.com/connect/settings/profile)

- [ ] [Customise brand settings](https://dashboard.stripe.com/settings/connect/stripe-dashboard/branding)
Add a business name, icon, and brand colour.

## 📌 Create a connected account

When a user (seller or service provider) signs up on your platform, create a user [Account](https://docs.stripe.com/api/accounts.md) (referred to as a *connected account*) so you can accept payments and move funds to their bank account. Connected accounts represent your users in Stripe’s API and facilitate the collection of information requirements so Stripe can verify the user’s identity. For a Q&A product that pays for answers, the connected account represents the writer.

> This guide uses Express accounts which have certain [restrictions](https://docs.stripe.com/connect/express-accounts.md#prerequisites-for-using-express). You can evaluate [Custom accounts](https://docs.stripe.com/connect/custom-accounts.md) as an alternative.

### Customize your signup form

In your [platform settings](https://dashboard.stripe.com/settings/connect/stripe-dashboard/branding), customise your Express sign-up form by changing the colour and logos that users see when they click your *Connect* (Connect is Stripe's solution for multi-party businesses, such as marketplace or software platforms, to route payments between sellers, customers, and other recipients) link.
![](https://b.stripecdn.com/docs-statics-srv/assets/oauth-form.4b13fc5edc56abd16004b4ccdff27fb6.png)

Default Express signup form
![](https://b.stripecdn.com/docs-statics-srv/assets/branding-settings-payouts.20c99c810389a4e7f5c55238e80a9fc8.png)

Branding settings

### Create a connected account link

You can create a connected account onboarding link by clicking **+Create** on the [Connected accounts](https://dashboard.stripe.com/connect/accounts) page, and selecting **Express** for the account type, along with the **transfers** capability. Click **Continue** to generate a link to share with the user you want to onboard.
![Create an account in the Dashboard](https://b.stripecdn.com/docs-statics-srv/assets/create-account-unified.450b8fb21ed13bcc165baa7db225e157.png)

Create a connected account
![](https://b.stripecdn.com/docs-statics-srv/assets/no-code-connect-express-link-unified.64f67a6c708c26fa52ec9b1ac1327b40.png)

Create an onboarding link

This link directs users to a form where they can provide information to connect to your platform. For example, if you have a Q&A platform, you can provide a link for writers to connect with the platform. The link is only for the single connected account you created. After your user completes the onboarding flow, you can view them in your accounts list.
![](https://b.stripecdn.com/docs-statics-srv/assets/dashboard-account-payout.94e15f1be4a11a54d18fc305433e50f4.png)

## 📌 Add funds to your balance

To add funds, go to the [Balance](https://dashboard.stripe.com/balance/overview) section in the Dashboard. Click **Add to balance** and select a balance to add to funds to.

Select **Payments balance** to add funds that are paid out to your connected accounts. You can also use funds added to the payments balance to cover future refunds and disputes or to repay your platform’s negative balance. To learn more about **Refunds and disputes balance**, see [adding funds to your Stripe balance](https://docs.stripe.com/get-started/account/add-funds.md).

### Verify your bank account

Go through the verification process in the Dashboard when you first attempt to add funds from an unverified bank account. If your bank account is unverified, you’ll need to confirm two microdeposits from Stripe. These deposits appear in your online banking statement within 1-2 business days. You’ll see `ACCTVERIFY` as the statement description.

Stripe notifies you in the Dashboard and through email when the microdeposits have arrived in your account. To complete the verification process, click the Dashboard notification in the [Balance](https://dashboard.stripe.com/balance/overview) section, enter the two microdeposit amounts, and click **Verify account**.
![](https://b.stripecdn.com/docs-statics-srv/assets/top-ups4.85d1f2d8440f525714d0f2d20775e2d1.png)

### Add funds

Once verified, use the [Dashboard](https://dashboard.stripe.com/balance/overview) to add funds to your account balance.

1. In the Dashboard, go to the [Balance](https://dashboard.stripe.com/balance/overview) section.
1. Click **Add to balance**, and then select **Payments balance**.
1. Enter the amount to top-up.
1. If applicable, select a payment method from the dropdown (bank debit, bank transfer, or wire transfer).
1. For bank debits, verify the amount and click **Add funds**. For bank transfers, use the Stripe banking information to initiate a bank transfer or wire transfer from your bank.
1. The resulting object is called a [top-up](https://docs.stripe.com/api/topups/object.md), which you can view in the Dashboard’s [Top-ups](https://dashboard.stripe.com/topups) section. For bank transfers, the top-up isn’t created until the funds are received.

### View funds

View your funds in the [Top-ups](https://dashboard.stripe.com/topups) tab under the [Balance](https://dashboard.stripe.com/balance/overview) page. Each time you add funds we create a `top-up` object with a unique ID with the following format: **tu\_XXXXXX**. You can see this in the top-up’s detailed view.

### Settlement timing

US platforms add funds through ACH debit and can take 5-6 business days to become available in your Stripe balance. You can request a review of your account for faster settlement timing by contacting [Stripe Support](https://support.stripe.com/contact).

As we learn more about your account, Stripe might be able to decrease your settlement timing automatically.

Adding funds for future refunds and disputes or to repay a negative balance can happen through [bank or electronic transfers](https://docs.stripe.com/get-started/account/add-funds.md) and are available in 1-2 business days.

## Pay out to your user

After your user completes [the onboarding process](https://docs.stripe.com/connect/add-and-pay-out-guide.md#create-connected-account) and you’ve added funds to your balance, you can transfer some of your balance to your connected accounts. In this example, money is transferred from the Q&A platform’s balance to the individual writer.

To pay your user, go to the **Balance** section of an account’s details page and click **Add funds**. By default, any funds you transfer to a connected account accumulate in the connected account’s Stripe balance and are paid out on a daily rolling basis. You can change the payout frequency by clicking the right-most button in the **Balance** section and selecting **Edit payout schedule**.
![](https://b.stripecdn.com/docs-statics-srv/assets/send-funds.5c34a4e2e038c3a5343c7aa165eb3787.png)

Send funds to user
![](https://b.stripecdn.com/docs-statics-srv/assets/edit-payout-schedule.537eca9bac08a738533bd644e9dd2280.png)

Edit payout schedule

## See also

- [Managing connected accounts in the Dashboard](https://docs.stripe.com/connect/dashboard.md)
