/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useContext, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import {
  API,
  getLogo,
  showError,
  showInfo,
  showSuccess,
  updateAPI,
  getSystemName,
  setUserData,
} from '../../helpers';
import Turnstile from 'react-turnstile';
import { Button, Card, Checkbox, Form, Modal } from '@douyinfe/semi-ui';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { IconMail, IconLock } from '@douyinfe/semi-icons';
import TwoFAVerification from './TwoFAVerification';
import { useTranslation } from 'react-i18next';

const LoginForm = () => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [inputs, setInputs] = useState({ username: '', password: '' });
  const { username, password } = inputs;
  const [searchParams] = useSearchParams();
  const [, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [loginLoading, setLoginLoading] = useState(false);
  const [resetPasswordLoading, setResetPasswordLoading] = useState(false);
  const [showTwoFA, setShowTwoFA] = useState(false);
  const [agreedToTerms, setAgreedToTerms] = useState(false);
  const [hasUserAgreement, setHasUserAgreement] = useState(false);
  const [hasPrivacyPolicy, setHasPrivacyPolicy] = useState(false);

  const logo = getLogo();
  const systemName = getSystemName();

  const affCode = new URLSearchParams(window.location.search).get('aff');
  if (affCode) {
    localStorage.setItem('aff', affCode);
  }

  const status = useMemo(() => {
    if (statusState?.status) return statusState.status;
    const savedStatus = localStorage.getItem('status');
    if (!savedStatus) return {};
    try {
      return JSON.parse(savedStatus) || {};
    } catch (err) {
      return {};
    }
  }, [statusState?.status]);

  useEffect(() => {
    if (status?.turnstile_check) {
      setTurnstileEnabled(true);
      setTurnstileSiteKey(status.turnstile_site_key);
    }
    setHasUserAgreement(status?.user_agreement_enabled || false);
    setHasPrivacyPolicy(status?.privacy_policy_enabled || false);
  }, [status]);

  useEffect(() => {
    if (searchParams.get('expired')) {
      showError(t('未登录或登录已过期，请重新登录'));
    }
  }, [searchParams, t]);

  function handleChange(name, value) {
    setInputs((prev) => ({ ...prev, [name]: value }));
  }

  async function handleSubmit() {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo('请稍后几秒重试，Turnstile 正在检查用户环境！');
      return;
    }
    if (!username || !password) {
      showError('请输入用户名和密码！');
      return;
    }
    setLoginLoading(true);
    try {
      const res = await API.post(`/api/user/login?turnstile=${turnstileToken}`, {
        username,
        password,
      });
      const { success, message, data } = res.data;
      if (success) {
        if (data && data.require_2fa) {
          setShowTwoFA(true);
          return;
        }
        userDispatch({ type: 'login', payload: data });
        setUserData(data);
        updateAPI();
        showSuccess('登录成功！');
        if (username === 'root' && password === '123456') {
          Modal.error({
            title: '您正在使用默认密码！',
            content: '请立刻修改默认密码！',
            centered: true,
          });
        }
        navigate('/console');
      } else {
        showError(message);
      }
    } catch (error) {
      showError('登录失败，请重试');
    } finally {
      setLoginLoading(false);
    }
  }

  const handleResetPasswordClick = () => {
    setResetPasswordLoading(true);
    navigate('/reset');
    setResetPasswordLoading(false);
  };

  const handle2FASuccess = (data) => {
    userDispatch({ type: 'login', payload: data });
    setUserData(data);
    updateAPI();
    showSuccess('登录成功！');
    navigate('/console');
  };

  const handleBackToLogin = () => {
    setShowTwoFA(false);
    setInputs({ username: '', password: '' });
  };

  return (
    <div className='relative overflow-hidden bg-gray-100 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8'>
      <div
        className='blur-ball blur-ball-indigo'
        style={{ top: '-80px', right: '-80px', transform: 'none' }}
      />
      <div
        className='blur-ball blur-ball-teal'
        style={{ top: '50%', left: '-120px' }}
      />
      <div className='w-full max-w-sm mt-[60px]'>
        <div className='flex flex-col items-center'>
          <div className='w-full max-w-md'>
            <div className='flex items-center justify-center mb-6 gap-2'>
              <img src={logo} alt='Logo' className='h-10 rounded-full' />
              <Title heading={3}>{systemName}</Title>
            </div>

            <Card className='border-0 !rounded-2xl overflow-hidden'>
              <div className='flex justify-center pt-6 pb-2'>
                <Title heading={3} className='text-gray-800 dark:text-gray-200'>
                  {t('登 录')}
                </Title>
              </div>
              <div className='px-2 py-8'>
                <Form className='space-y-3'>
                  <Form.Input
                    field='username'
                    label={t('用户名或邮箱')}
                    placeholder={t('请输入您的用户名或邮箱地址')}
                    name='username'
                    onChange={(value) => handleChange('username', value)}
                    prefix={<IconMail />}
                  />
                  <Form.Input
                    field='password'
                    label={t('密码')}
                    placeholder={t('请输入您的密码')}
                    name='password'
                    mode='password'
                    onChange={(value) => handleChange('password', value)}
                    prefix={<IconLock />}
                  />

                  {(hasUserAgreement || hasPrivacyPolicy) && (
                    <div className='pt-4'>
                      <Checkbox
                        checked={agreedToTerms}
                        onChange={(e) => setAgreedToTerms(e.target.checked)}
                      >
                        <Text size='small' className='text-gray-600'>
                          {t('我已阅读并同意')}
                          {hasUserAgreement && (
                            <a
                              href='/user-agreement'
                              target='_blank'
                              rel='noopener noreferrer'
                              className='text-blue-600 hover:text-blue-800 mx-1'
                            >
                              {t('用户协议')}
                            </a>
                          )}
                          {hasUserAgreement && hasPrivacyPolicy && t('和')}
                          {hasPrivacyPolicy && (
                            <a
                              href='/privacy-policy'
                              target='_blank'
                              rel='noopener noreferrer'
                              className='text-blue-600 hover:text-blue-800 mx-1'
                            >
                              {t('隐私政策')}
                            </a>
                          )}
                        </Text>
                      </Checkbox>
                    </div>
                  )}

                  <div className='space-y-2 pt-2'>
                    <Button
                      theme='solid'
                      className='w-full !rounded-full'
                      type='primary'
                      htmlType='submit'
                      onClick={handleSubmit}
                      loading={loginLoading}
                      disabled={
                        (hasUserAgreement || hasPrivacyPolicy) &&
                        !agreedToTerms
                      }
                    >
                      {t('继续')}
                    </Button>
                    <Button
                      theme='borderless'
                      type='tertiary'
                      className='w-full !rounded-full'
                      onClick={handleResetPasswordClick}
                      loading={resetPasswordLoading}
                    >
                      {t('忘记密码？')}
                    </Button>
                  </div>
                </Form>

                <div className='mt-6 text-center text-sm'>
                  <Text>
                    {t('没有账户？')}{' '}
                    <Link
                      to='/register'
                      className='text-blue-600 hover:text-blue-800 font-medium'
                    >
                      {t('注册')}
                    </Link>
                  </Text>
                </div>
              </div>
            </Card>
          </div>
        </div>

        <Modal
          title='两步验证'
          visible={showTwoFA}
          onCancel={handleBackToLogin}
          footer={null}
          width={450}
          centered
        >
          <TwoFAVerification
            onSuccess={handle2FASuccess}
            onBack={handleBackToLogin}
            isModal={true}
          />
        </Modal>

        {turnstileEnabled && (
          <div className='flex justify-center mt-6'>
            <Turnstile
              sitekey={turnstileSiteKey}
              onVerify={(token) => setTurnstileToken(token)}
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default LoginForm;
