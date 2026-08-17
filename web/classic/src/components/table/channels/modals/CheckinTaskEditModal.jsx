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

import React, { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Modal } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../../helpers';

const defaultTask = {
  name: '',
  status: 1,
  request_url: '',
  request_method: 'POST',
  auth_type: 'system_access_token',
  api_user: '',
  secret: '',
  proxy_url: '',
  clear_proxy: false,
  timeout_seconds: 20,
  retry_count: 1,
  interval_minutes: 1440,
};

const buildFormValues = (editingTask) => {
  if (!editingTask) return defaultTask;
  return {
    name: editingTask.name ?? '',
    status: editingTask.status ?? 1,
    request_url: editingTask.request_url ?? '',
    request_method: editingTask.request_method || 'POST',
    auth_type: editingTask.auth_type || 'system_access_token',
    api_user: editingTask.api_user ?? '',
    secret: '',
    proxy_url: '',
    clear_proxy: false,
    timeout_seconds: editingTask.timeout_seconds ?? 20,
    retry_count: editingTask.retry_count ?? 1,
    interval_minutes: editingTask.interval_minutes ?? 1440,
  };
};

const CheckinTaskEditModal = ({ visible, editingTask, onCancel, onSaved }) => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [saving, setSaving] = useState(false);
  const isEdit = !!editingTask;

  const submitTask = async () => {
    let values;
    try {
      // Semi 的 validate() 校验通过时 resolve 表单值，失败时 reject；错误提示由 Form 展示。
      values = await formApiRef.current?.validate();
    } catch {
      return;
    }
    if (!values) return;
    setSaving(true);
    try {
      const payload = {
        name: values.name,
        status: values.status,
        request_url: values.request_url,
        request_method: values.request_method,
        auth_type: values.auth_type,
        api_user: values.api_user,
        secret: values.secret ?? '',
        proxy_url: values.proxy_url ?? '',
        clear_proxy: !!values.clear_proxy,
        timeout_seconds: values.timeout_seconds,
        retry_count: values.retry_count,
        interval_minutes: values.interval_minutes,
      };
      const res = isEdit
        ? await API.put('/api/channel-checkin-task/', {
            ...payload,
            id: editingTask.id,
          })
        : await API.post('/api/channel-checkin-task/', payload);
      if (!res.data?.success) {
        showError(res.data?.message || t('保存签到任务失败'));
        return;
      }
      showSuccess(t('保存成功'));
      onSaved?.(res.data.data);
    } catch (error) {
      showError(error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      visible={visible}
      title={isEdit ? t('编辑签到任务') : t('新增签到任务')}
      onCancel={onCancel}
      onOk={submitTask}
      confirmLoading={saving}
      okText={t('保存')}
      style={{ maxWidth: 680 }}
    >
      <Form
        key={editingTask?.id || 'new'}
        initValues={buildFormValues(editingTask)}
        getFormApi={(api) => (formApiRef.current = api)}
        labelPosition='top'
      >
        <Form.Input
          field='name'
          label={t('任务名称')}
          rules={[{ required: true, message: t('请输入任务名称') }]}
        />
        <Form.Select
          field='status'
          label={t('状态')}
          optionList={[
            { value: 1, label: t('启用') },
            { value: 0, label: t('停用') },
          ]}
        />
        <Form.Input
          field='request_url'
          label={t('签到请求地址')}
          placeholder='https://example.com/api/user/checkin'
          rules={[
            { required: true, message: t('请输入 HTTPS 签到请求地址') },
            {
              validator: (rule, value) =>
                !value || value.trim().startsWith('https://'),
              message: t('签到请求地址必须为 HTTPS 地址'),
            },
          ]}
        />
        <Form.Select
          field='request_method'
          label={t('请求方法')}
          optionList={[
            { value: 'POST', label: 'POST' },
            { value: 'GET', label: 'GET' },
          ]}
        />
        <Form.Select
          field='auth_type'
          label={t('认证方式')}
          optionList={[
            { value: 'system_access_token', label: t('系统访问令牌') },
            { value: 'cookie', label: t('Cookie') },
          ]}
        />
        <Form.Input
          field='api_user'
          label={t('用户 ID')}
          rules={[{ required: true, message: t('请输入用户 ID') }]}
        />
        <Form.Input
          field='secret'
          mode='password'
          label={isEdit ? t('更新认证凭据（留空则保持不变）') : t('认证凭据')}
          rules={
            isEdit ? [] : [{ required: true, message: t('请输入认证凭据') }]
          }
        />
        <Form.Input
          field='proxy_url'
          mode='password'
          label={t('固定代理地址（可选）')}
          placeholder='https://proxy.example.com:8443'
          extraText={
            editingTask?.has_proxy_url
              ? t('已保存固定代理；留空则保持不变。')
              : t('仅用于常规网络连通性；不支持代理池、轮换或自动切换。')
          }
        />
        {editingTask?.has_proxy_url && (
          <Form.Switch field='clear_proxy' label={t('清除已保存的固定代理')} />
        )}
        <Form.InputNumber
          field='timeout_seconds'
          label={t('请求超时（秒）')}
          min={5}
          max={120}
        />
        <Form.InputNumber
          field='retry_count'
          label={t('网络失败重试次数')}
          min={0}
          max={2}
          extraText={t(
            '仅对网络或服务暂时不可用重试；验证码、403 和风控响应不会重试。',
          )}
        />
        <Form.InputNumber
          field='interval_minutes'
          label={t('执行间隔（分钟）')}
          min={60}
          extraText={t('最短 60 分钟；建议每日执行一次。')}
        />
      </Form>
    </Modal>
  );
};

export default CheckinTaskEditModal;
