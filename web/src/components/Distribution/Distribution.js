/* eslint-disable no-alert */

import React from 'react';
import PropTypes from 'prop-types';
import styled from 'styled-components';
import moment from 'moment';
import Helmet from 'react-helmet';
import { Flex, Box } from 'grid-styled';
import { FormattedMessage, FormattedHTMLMessage, injectIntl } from 'react-intl';
import { rem } from 'polished';
import { COLORS, SPACE, BOX_SHADOWS, BORDER_RADIUS } from 'config';

import Button from 'components/Button';
import Container from 'components/Container';
import Footer from 'components/Footer';
import Header from 'components/Header';
import Heading from 'components/Heading';
import Input from 'components/Input';
import Modal, { styles } from 'components/Modal';
import Text from 'components/Text';
import media from '../../utils/media';
import Select from 'react-select';
import 'react-select/dist/react-select.css';
import QRCode from 'qrcode.react';

import { checkStatus, getAddress, getConfig, checkExchangeStatus } from '../../utils/distributionAPI';

const Wrapper = styled.div`
  background-color: ${COLORS.gray[1]};
  padding: ${rem(SPACE[5])} 0;

  ${media.md.css`
    padding: ${rem(SPACE[7])} 0;
  `}
`;

const Address = Heading.extend`
  word-break: break-all;
  background-color: ${COLORS.gray[0]};
  border-radius: ${BORDER_RADIUS.base};
  box-shadow: ${BOX_SHADOWS.base};
  padding: 1rem;
`;

class Distribution extends React.Component {
  constructor() {
    super();
    this.state = {
      status: [],
      coinType: 'SKY',
      mdlAddress: null,
      btcAddress: '',
      statusIsOpen: false,
      addressLoading: false,
      statusLoading: false,
    };

    this.handleChange = this.handleChange.bind(this);
    this.handleCoinTypeChange = this.handleCoinTypeChange.bind(this);
    this.getAddress = this.getAddress.bind(this);
    this.checkStatus = this.checkStatus.bind(this);
    this.closeModals = this.closeModals.bind(this);
    this.checkExchangeStatus = this.checkExchangeStatus.bind(this);
  }

  componentDidMount() {
    this.getConfig().then(() => this.checkExchangeStatus());
  }

  checkExchangeStatus() {
    return checkExchangeStatus()
    .then(status => {
      if (status.error !== "") {
        this.setState({
          disabledReason: "coinsSoldOut",
          balance: status.balance,
          enabled: false
        });
      } else {
        this.setState({
          balance: status.balance
        });
      }
    });
  }

  getConfig() {
    return getConfig().then(config => this.setState({ ...config }));
  }

  getAddress() {
    if (!this.state.mdlAddress) {
      return alert(
        this.props.intl.formatMessage({
          id: 'distribution.errors.noSkyAddress',
        }),
      );
    }

    this.setState({
      addressLoading: true,
    });

    return getAddress(this.state)
      .then((res) => {
        this.setState({
          btcAddress: res,
        });
      })
      .catch((err) => {
        alert(err.message);
      })
      .then(() => {
        this.setState({
          addressLoading: false,
        });
      });
  }

  handleChange(event) {
    this.setState({
      mdlAddress: event.target.value
    });
  }


  closeModals() {
    this.setState({
      statusIsOpen: false,
    });
  }

  currentCoinPrice() {
    switch (this.state.coinType) {
      case "BTC": return this.state.supported[0].exchange_rate;
      case "ETH": return this.state.supported[1].exchange_rate;
      case "SKY": return this.state.supported[2].exchange_rate;
      case "WAVES": return this.state.supported[3].exchange_rate;
      default: return "1";
    }
  }

  handleCoinTypeChange(selectedOption) {
    this.setState({
      btcAddress: '',
      coinType: selectedOption.value,
    });
  }

  checkStatus() {
    if (!this.state.mdlAddress) {
      return alert(
        this.props.intl.formatMessage({
          id: 'distribution.errors.noSkyAddress',
        }),
      );
    }

    this.setState({
      statusLoading: true,
    });

    return checkStatus(this.state.mdlAddress)
      .then((res) => {
        this.setState({
          statusIsOpen: true,
          status: res,
        });
      })
      .catch((err) => {
        alert(err.message);
      })
      .then(() => {
        this.setState({
          statusLoading: false,
        });
        return this.checkExchangeState();
      });
  }

  render() {
    const { intl } = this.props;
    return (
      <div>
        <Helmet>
          <title>{intl.formatMessage({ id: 'distribution.title' })}</title>
        </Helmet>

        <Header external />

        <Wrapper>
          <Modal
            contentLabel="Status"
            style={styles}
            isOpen={this.state.statusIsOpen}
            onRequestClose={this.closeModals}
          >
            <Heading heavy color="black" fontSize={[2, 3]} my={[3, 5]}>
              <FormattedMessage
                id="distribution.statusFor"
                values={{
                  mdlAddress: this.state.mdlAddress,
                }}
              />
            </Heading>

            <Text as="div" color="black" fontSize={[2, 3]} my={[3, 5]}>
              {this.state.status.map(status => (
                <p key={status.seq}>
                  <FormattedMessage
                    id={`distribution.statuses.${status.status}`}
                    values={{
                      id: String(status.seq),
                      updated: moment.unix(status.updated_at).locale(intl.locale).format('LL LTS'),
                    }}
                  />
                </p>
              ))}
            </Text>
          </Modal>

          <Container>
            {!this.state.enabled ? <Flex column>

              <Heading heavy as="h2" fontSize={[5, 6]} color="black" mb={[4, 6]}>
                {(this.state.disabledReason === "coinsSoldOut") ?
                 <FormattedMessage id="distribution.errors.coinsSoldOut" /> :
                 <FormattedMessage id="distribution.headingEnded" />}
              </Heading>
              <Text heavy color="black" fontSize={[2, 3]} as="div">
                <FormattedHTMLMessage id="distribution.ended" />
              </Text>
            </Flex> : <Flex justify="center">
              <Box width={[1 / 1, 1 / 1, 2 / 3]} py={[5, 7]}>
                <Heading heavy as="h2" fontSize={[5, 6]} color="black" mb={[4, 6]}>
                  <FormattedMessage id="distribution.heading" />
                </Heading>
                <Text heavy color="black" fontSize={[2, 3]} mb={[4, 6]} as="div">
                  <FormattedHTMLMessage
                    id="distribution.inventory"
                    values={{
                      coins: this.state.balance && this.state.balance.coins,
                    }}
                  />
                </Text>

                <Text heavy color="black" fontSize={[2, 3]} as="div">
                  <FormattedHTMLMessage id="distribution.instructions"
                  values={{max_bound_addrs:this.state.max_bound_addrs, coinType:this.state.coinType}}/>
                </Text>

                <Input
                 as="div"
                  placeholder={intl.formatMessage({ id: 'distribution.enterAddress' })}
                  value={this.state.address}
                  onChange={this.handleChange}
                />

                <div>
                  <Select
                    name="coin_type"
                    value={this.state.coinType}
                    clearable={false}
                    onChange={this.handleCoinTypeChange}
                    options={[
                      { value: 'BTC', label: 'Bitcoin', disabled: !this.state.supported[0].enabled },
                      { value: 'ETH', label: 'Ethereum', disabled: !this.state.supported[1].enabled  },
                      { value: 'SKY', label: 'Skycoin (SKY) -- www.skycoin.com --', disabled: !this.state.supported[2].enabled },
                      { value: 'WAVES', label: 'Waves (Disabled)', disabled: !this.state.supported[3].enabled },
                      // { value: 'MDL.life', label: 'MDL.life (pre-MDL token on Waves)', disabled: this.state.supported[4].enabled },
                    ]}
                  />
                  <Text heavy color="grey" fontSize={[2, 3]}>
                    <FormattedMessage
                      id="distribution.rate"
                      values={{
                        rate: parseFloat(this.currentCoinPrice()),
                        rateRev: parseFloat(1/this.currentCoinPrice()).toPrecision(3),
                        coinType: this.state.coinType,
                      }}
                    />
                  </Text>
                </div>

                {this.state.btcAddress && <Address heavy color="black" fontSize={[2, 3]} as="p">
                  <strong><FormattedHTMLMessage id="distribution.recAddress" values={{coinType:this.state.coinType}}/>:<br/></strong>
                  <center>
                  {this.state.btcAddress}
                  <div>
                    <br/>
                  <QRCode value={this.state.btcAddress} renderAs='svg'/>
                    <br/>
                  </div>
                  </center>
                </Address>}

                <div>
                  <Button
                    big
                    onClick={this.getAddress}
                    color="white"
                    bg="base"
                    mr={[2, 5]}
                    fontSize={[1, 3]}
                  >
                    {this.state.addressLoading
                      ? <FormattedMessage id="distribution.loading" />
                      : <FormattedMessage id="distribution.getAddress" />}
                  </Button>

                  <Button
                    onClick={this.checkStatus}
                    color="base"
                    big
                    outlined
                    fontSize={[1, 3]}
                  >
                    {this.state.statusLoading
                      ? <FormattedMessage id="distribution.loading" />
                      : <FormattedMessage id="distribution.checkStatus" />}
                  </Button>
                </div>
              </Box>
            </Flex>}
          </Container>
        </Wrapper>

        <Footer external />
      </div>
    );
  }
}

Distribution.propTypes = {
  intl: PropTypes.shape({
    formatMessage: PropTypes.func.isRequired,
  }).isRequired,
};

export default injectIntl(Distribution);
