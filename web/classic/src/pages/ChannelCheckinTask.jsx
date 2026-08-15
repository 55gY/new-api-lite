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

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Descriptions,
  Form,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconEdit,
  IconHistory,
  IconPlay,
  IconPlus,
  IconRefresh,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../helpers';

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

const formatTimestamp = (timestamp) => {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleString();
};

const ChannelCheckinTask = () => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editingTask, setEditingTask] = useState(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [logs, setLogs] = useState([]);
  const [logsVisible, setLogsVisible] = useState(false);
  const [logTaskName, setLogTaskName] = useState('');

  const loadTasks = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/channel-checkin-task/');
      if (res.data?.success) {
        setTasks(res.data.data?.items || []);
      } else {
        showError(res.data?.message || t('获取签到任务失败'));
      }
    } catch (error) {
      showError(error);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadTasks();
  }, [loadTasks]);

  const openCreate = () => {
    setEditingTask(null);
    setModalVisible(true);
  };

  const openEdit = (task) => {
    setEditingTask(task);
    setModalVisible(true);
  };

  const submitTask = async () => {
    const values = formApiRef.current?.getValues();
    if (!values) return;
    const errors = await formApiRef.current?.validate();
    if (errors) return;
    setSaving(true);
    try {
      const payload = { ...values };
      if (editingTask) payload.id = editingTask.id;
      const res = editingTask
        ? await API.put('/api/channel-checkin-task/', payload)
        : await API.post('/api/channel-checkin-task/', payload);
      if (!res.data?.success) {
        showError(res.data?.message || t('保存签到任务失败'));
        return;
      }
      showSuccess(t('保存成功'));
      setModalVisible(false);
      await loadTasks();
    } catch (error) {
      showError(error);
    } finally {
      setSaving(false);
    }
  };

  const deleteTask = async (task) => {
    try {
      const res = await API.delete(`/api/channel-checkin-task/${task.id}`);
      if (!res.data?.success) {
        showError(res.data?.message || t('删除签到任务失败'));
        return;
      }
      showSuccess(t('删除成功'));
      await loadTasks();
    } catch (error) {
      showError(error);
    }
  };

  const runTask = async (task) => {
    try {
      const res = await API.post(`/api/channel-checkin-task/${task.id}/run`);
      if (res.data?.success) {
        showSuccess(res.data.message || t('签到任务已执行'));
      } else {
        showError(res.data?.message || t('签到任务需要人工处理'));
      }
      await loadTasks();
    } catch (error) {
      showError(error);
    }
  };

  const openLogs = async (task) => {
    try {
      const res = await API.get(`/api/channel-checkin-task/${task.id}/logs`);
      if (!res.data?.success) {
        showError(res.data?.message || t('获取执行日志失败'));
        return;
      }
      setLogs(res.data.data?.items || []);
      setLogTaskName(task.name);
      setLogsVisible(true);
    } catch (error) {
      showError(error);
    }
  };

  const statusTag = (task) => {
    if (task.last_run_status === 'manual_action_required') {
      return <Tag color='orange'>{t('需人工处理')}</Tag>;
    }
    if (task.last_run_status === 'authentication_failed') {
      return <Tag color='red'>{t('凭据失效')}</Tag>;
    }
    if (task.last_run_status === 'configuration_error') {
      return <Tag color='red'>{t('配置异常')}</Tag>;
    }
    if (task.status !== 1) {
      return <Tag color='grey'>{t('已停用')}</Tag>;
    }
    if (task.last_run_status === 'success') {
      return <Tag color='green'>{t('正常')}</Tag>;
    }
    return <Tag color='blue'>{t('已启用')}</Tag>;
  };

  const columns = [
    { title: t('名称'), dataIndex: 'name', width: 180 },
    {
      title: t('状态'),
      width: 110,
      render: (_, task) => statusTag(task),
    },
    {
      title: t('认证方式'),
      dataIndex: 'auth_type',
      width: 140,
      render: (value) => (value === 'cookie' ? t('Cookie') : t('系统访问令牌')),
    },
    {
      title: t('计划间隔'),
      dataIndex: 'interval_minutes',
      width: 120,
      render: (value) => t('{{count}} 分钟', { count: value }),
    },
    {
      title: t('上次执行'),
      width: 180,
      render: (_, task) => formatTimestamp(task.last_run_at),
    },
    {
      title: t('下次执行'),
      width: 180,
      render: (_, task) =>
        task.status === 1 ? formatTimestamp(task.next_run_at) : '-',
    },
    {
      title: t('结果'),
      dataIndex: 'last_run_message',
      render: (value) => value || '-',
    },
    {
      title: t('操作'),
      width: 250,
      render: (_, task) => (
        <Space>
          <Button
            theme='borderless'
            icon={<IconPlay />}
            onClick={() => runTask(task)}
            disabled={task.status !== 1}
          >
            {t('执行')}
          </Button>
          <Button
            theme='borderless'
            icon={<IconHistory />}
            onClick={() => openLogs(task)}
          >
            {t('日志')}
          </Button>
          <Button
            theme='borderless'
            icon={<IconEdit />}
            onClick={() => openEdit(task)}
          >
            {t('编辑')}
          </Button>
          <Popconfirm
            title={t('确认删除该签到任务？关联渠道将自动解除关联。')}
            onConfirm={() => deleteTask(task)}
          >
            <Button theme='borderless' type='danger' icon={<IconDelete />}>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const taskFormValues = editingTask
    ? {
        ...editingTask,
        secret: '',
        proxy_url: '',
        clear_proxy: false,
      }
    : defaultTask;

  return (
    <div className='mt-[60px] px-2'>
      <Card
        title={t('签到任务')}
        headerExtraContent={
          <Space>
            <Button
              icon={<IconRefresh />}
              onClick={loadTasks}
              loading={loading}
            >
              {t('刷新')}
            </Button>
            <Button type='primary' icon={<IconPlus />} onClick={openCreate}>
              {t('新增签到任务')}
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type='tertiary'>
          {t(
            '任务仅向管理员配置的 HTTPS 地址发起签到请求。遇到验证码、风控或拦截响应时会自动停用并等待人工处理，不会尝试绕过站点防护。',
          )}
        </Typography.Paragraph>
        <Table
          rowKey='id'
          columns={columns}
          dataSource={tasks}
          loading={loading}
          pagination={false}
          empty={t('暂无签到任务')}
        />
      </Card>

      <Modal
        visible={modalVisible}
        title={editingTask ? t('编辑签到任务') : t('新增签到任务')}
        onCancel={() => setModalVisible(false)}
        onOk={submitTask}
        confirmLoading={saving}
        okText={t('保存')}
        style={{ maxWidth: 680 }}
      >
        <Form
          key={editingTask?.id || 'new'}
          initValues={taskFormValues}
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
            label={
              editingTask ? t('更新认证凭据（留空则保持不变）') : t('认证凭据')
            }
            rules={
              editingTask
                ? []
                : [{ required: true, message: t('请输入认证凭据') }]
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
            <Form.Switch
              field='clear_proxy'
              label={t('清除已保存的固定代理')}
            />
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

      <Modal
        visible={logsVisible}
        title={t('{{name}} - 执行日志', { name: logTaskName })}
        footer={null}
        onCancel={() => setLogsVisible(false)}
        style={{ maxWidth: 820 }}
      >
        <Descriptions>
          <Descriptions.Item itemKey='notice'>
            {t('日志不包含 Cookie、令牌、代理凭据或原始响应内容。')}
          </Descriptions.Item>
        </Descriptions>
        <Table
          rowKey='id'
          pagination={false}
          dataSource={logs}
          columns={[
            {
              title: t('时间'),
              dataIndex: 'created_at',
              render: formatTimestamp,
            },
            { title: t('状态'), dataIndex: 'status' },
            { title: t('HTTP 状态'), dataIndex: 'http_status' },
            { title: t('消息'), dataIndex: 'message' },
          ]}
        />
      </Modal>
    </div>
  );
};

export default ChannelCheckinTask;
