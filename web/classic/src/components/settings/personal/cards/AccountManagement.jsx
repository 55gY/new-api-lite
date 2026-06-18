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

import React from 'react';
import { Button, Card, Input, Space, Typography, Avatar } from '@douyinfe/semi-ui';
import { IconKey, IconLock, IconDelete } from '@douyinfe/semi-icons';
import { ShieldCheck } from 'lucide-react';
import TwoFASetting from '../components/TwoFASetting';

const AccountManagement = ({
  t,
  systemToken,
  generateAccessToken,
  handleSystemTokenClick,
  setShowChangePasswordModal,
  setShowAccountDeleteModal,
}) => {
  return (
    <Card className='!rounded-2xl'>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='teal' className='mr-3 shadow-md'>
          <ShieldCheck size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium'>
            {t('账户管理')}
          </Typography.Text>
          <div className='text-xs text-gray-600'>
            {t('安全设置和身份验证')}
          </div>
        </div>
      </div>

      <div className='py-4'>
        <div className='space-y-6'>
          <Space vertical className='w-full'>
            <Card className='!rounded-xl w-full'>
              <div className='flex flex-col sm:flex-row items-start sm:justify-between gap-4'>
                <div className='flex items-start w-full sm:w-auto'>
                  <div className='w-12 h-12 rounded-full bg-slate-100 flex items-center justify-center mr-4 flex-shrink-0'>
                    <IconKey size='large' className='text-slate-600' />
                  </div>
                  <div className='flex-1'>
                    <Typography.Title heading={6} className='mb-1'>
                      {t('系统访问令牌')}
                    </Typography.Title>
                    <Typography.Text type='tertiary' className='text-sm'>
                      {t('用于API调用的身份验证令牌，请妥善保管')}
                    </Typography.Text>
                    {systemToken && (
                      <div className='mt-3'>
                        <Input
                          readonly
                          value={systemToken}
                          onClick={handleSystemTokenClick}
                          size='large'
                          prefix={<IconKey />}
                        />
                      </div>
                    )}
                  </div>
                </div>
                <Button
                  type='primary'
                  theme='solid'
                  onClick={generateAccessToken}
                  className='!bg-slate-600 hover:!bg-slate-700 w-full sm:w-auto'
                  icon={<IconKey />}
                >
                  {systemToken ? t('重新生成') : t('生成令牌')}
                </Button>
              </div>
            </Card>

            <Card className='!rounded-xl w-full'>
              <div className='flex flex-col sm:flex-row items-start sm:justify-between gap-4'>
                <div className='flex items-start w-full sm:w-auto'>
                  <div className='w-12 h-12 rounded-full bg-slate-100 flex items-center justify-center mr-4 flex-shrink-0'>
                    <IconLock size='large' className='text-slate-600' />
                  </div>
                  <div>
                    <Typography.Title heading={6} className='mb-1'>
                      {t('密码管理')}
                    </Typography.Title>
                    <Typography.Text type='tertiary' className='text-sm'>
                      {t('定期更改密码可以提高账户安全性')}
                    </Typography.Text>
                  </div>
                </div>
                <Button
                  type='primary'
                  theme='solid'
                  onClick={() => setShowChangePasswordModal(true)}
                  className='!bg-slate-600 hover:!bg-slate-700 w-full sm:w-auto'
                  icon={<IconLock />}
                >
                  {t('修改密码')}
                </Button>
              </div>
            </Card>

            <TwoFASetting t={t} />

            <Card className='!rounded-xl w-full'>
              <div className='flex flex-col sm:flex-row items-start sm:justify-between gap-4'>
                <div className='flex items-start w-full sm:w-auto'>
                  <div className='w-12 h-12 rounded-full bg-slate-100 flex items-center justify-center mr-4 flex-shrink-0'>
                    <IconDelete size='large' className='text-slate-600' />
                  </div>
                  <div>
                    <Typography.Title
                      heading={6}
                      className='mb-1 text-slate-700'
                    >
                      {t('删除账户')}
                    </Typography.Title>
                    <Typography.Text type='tertiary' className='text-sm'>
                      {t('此操作不可逆，所有数据将被永久删除')}
                    </Typography.Text>
                  </div>
                </div>
                <Button
                  type='danger'
                  theme='solid'
                  onClick={() => setShowAccountDeleteModal(true)}
                  className='w-full sm:w-auto !bg-slate-500 hover:!bg-slate-600'
                  icon={<IconDelete />}
                >
                  {t('删除账户')}
                </Button>
              </div>
            </Card>
          </Space>
        </div>
      </div>
    </Card>
  );
};

export default AccountManagement;
