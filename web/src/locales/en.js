export default {
  header: {
    navigation: {
      whitepapers: 'Whitepapers',
      downloads: 'Downloads',
      explorer: 'Explorer',
      blog: 'Blog',
      roadmap: 'Roadmap',
    },
  },
  footer: {
    getStarted: 'Get started',
    explore: 'Explore',
    community: 'Community',
    about: 'About',
    wallet_android: 'Android Wallet',
    wallet_mac: 'MacOS Wallet',
    wallet_win: 'Windows Wallet',
    wallet_lin: 'Linux Wallet',
    price: 'Coin Paprika',
    infographics: 'Infographics',
    explorer: 'Explorer',
    whitepapers: 'Whitepapers',
    blog: 'Blog',
    twitter: 'Twitter',
    reddit: 'Reddit',
    github: 'GitHub',
    telegram: 'Telegram',
    slack: 'Slack',
    roadmap: 'Roadmap',
    platform: 'Platform',
    skyMessenger: 'Sky-Messenger',
    cxPlayground: 'CX Playground',
    team: 'Team',
    subscribe: 'Mailing List',
    market: 'Markets',
    bitcoinTalks: 'Bitcointalks ANN',
    instagram: 'Instagram',
    facebook: 'Facebook',
    discord: 'Discord',
  },
  distribution: {
    rate: 'Current rate: 1 {coinType} = {rate} MDL, 1 MDL = {rateRev} {coinType}',
    inventory: 'Available Now: <strong>{coins} MDL</strong>',
    title: 'MDL Talent Hub Teller',
    heading: 'MDL Talent Hub Teller',
    headingEnded: 'MDL Talent Hub Teller is currently disabled',
    ended: `<p>Join the <a href="https://t.me/MDL_Talent_Hub" target="_blank">MDL Telegram</a>
      or follow the
      <a href="https://twitter.com/mdl_talent_hub" target="_blank">MDL Twitter</a>.

       <p>You can check the current market value of <a href="https://coinpaprika.com/coin/mdl-mdl">MDL at CoinPaprika</a>.</p>`,
    instructions: `<p>You can check the current market value of <a href="https://coinpaprika.com/coin/mdl-mdl">MDL at CoinPaprika</a>.</p>

<p>To purchase MDL tokens:</p>

<ul>
  <li>Enter your MDL address below</li>
  <li>Choose preferred payment method<br>(SKY, ETH, BTC - some might be temporarily disabled)</li>
  <li>Press Get Address <br>You&apos;ll receive a unique {coinType} address to purchase MDL</li>
  <li>Send funds to the provided {coinType} address</li>
</ul>

<p>Check the status of your order by entering your address and selecting <strong>Check status</strong>.</p>
<p>Each time you select <strong>Get Address</strong>, a new {coinType} address is generated.</p>
<p>A single MDL address can have up to {max_bound_addrs} {coinType} addresses assigned to it.</p>
    `,
    statusFor: 'Status for {mdlAddress}',
    enterAddress: 'Enter MDL address',
    getAddress: 'Get address',
    checkStatus: 'Check status',
    loading: 'Loading...',
    recAddress: '{coinType} address',
    errors: {
      noSkyAddress: 'Please enter your MDL address.',
      coinsSoldOut: 'All MDL tokens are currently sold out, check back later.',
    },
    statuses: {
      waiting_deposit: '[tx-{id} {updated}] Waiting for deposit.',
      waiting_send: '[tx-{id} {updated}] Deposit confirmed. Transaction is queued.',
      waiting_confirm: '[tx-{id} {updated}] MDL transaction sent.  Waiting to confirm.',
      done: '[tx-{id} {updated}] Completed. Check your MDL wallet.',
    },
  },
};
