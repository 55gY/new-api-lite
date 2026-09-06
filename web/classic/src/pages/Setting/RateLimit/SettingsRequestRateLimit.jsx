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

import React, { useEffect, useState, useRef } from 'react';
import { Button, Col, Form, Row, Spin } from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function RequestRateLimit(props) {
  const { t } = useTranslation();

  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    GlobalApiRateLimitEnable: false,
    GlobalApiRateLimitNum: 180,
    GlobalApiRateLimitDuration: 180,
    GlobalWebRateLimitEnable: true,
    GlobalWebRateLimitNum: 60,
    GlobalWebRateLimitDuration: 180,
    CriticalRateLimitEnable: true,
    CriticalRateLimitNum: 20,
    CriticalRateLimitDuration: 1200,
    SearchRateLimitEnable: true,
    SearchRateLimitNum: 10,
    SearchRateLimitDuration: 60,
    UploadRateLimitNum: 10,
    UploadRateLimitDuration: 60,
    DownloadRateLimitNum: 10,
    DownloadRateLimitDuration: 60,
    EmailVerificationMaxRequests: 2,
    EmailVerificationDuration: 30,
    ModelRequestRateLimitGroup: '{}',
    ModelRequestRateLimitEnabled: false,
    ModelRequestRateLimitCount: -1,
    ModelRequestRateLimitSuccessCount: 1000,
    ModelRequestRateLimitDurationMinutes: 1,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  const updateField = (field) => (value) => {
    setInputs((current) => ({ ...current, [field]: value }));
  };

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        value = inputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }

        for (let i = 0; i < res.length; i++) {
          if (!res[i].data.success) {
            return showError(res[i].data.message);
          }
        }

        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('全局 API 速率限制')}>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field='GlobalApiRateLimitEnable'
                  label={t('启用全局 API 速率限制')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  extraText={t('默认关闭；开启后按客户端 IP 限制 API 请求频率')}
                  onChange={(value) =>
                    updateField('GlobalApiRateLimitEnable')(value)
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('每周期最多请求次数')}
                  step={1}
                  min={1}
                  max={100000000}
                  suffix={t('次')}
                  field='GlobalApiRateLimitNum'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      GlobalApiRateLimitNum: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('限制周期')}
                  step={1}
                  min={1}
                  suffix={t('秒')}
                  field='GlobalApiRateLimitDuration'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      GlobalApiRateLimitDuration: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存全局 API 速率限制')}
              </Button>
            </Row>
          </Form.Section>

          <Form.Section text={t('全局 Web 速率限制')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field='GlobalWebRateLimitEnable'
                  label={t('启用全局 Web 速率限制')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={updateField('GlobalWebRateLimitEnable')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='GlobalWebRateLimitNum'
                  label={t('每周期最多请求次数')}
                  min={1}
                  max={100000000}
                  suffix={t('次')}
                  onChange={updateField('GlobalWebRateLimitNum')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='GlobalWebRateLimitDuration'
                  label={t('限制周期')}
                  min={1}
                  max={1200}
                  suffix={t('秒')}
                  onChange={updateField('GlobalWebRateLimitDuration')}
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存全局 Web 速率限制')}
              </Button>
            </Row>
          </Form.Section>

          <Form.Section text={t('关键接口与搜索速率限制')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field='CriticalRateLimitEnable'
                  label={t('启用关键接口速率限制')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={updateField('CriticalRateLimitEnable')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='CriticalRateLimitNum'
                  label={t('关键接口每周期次数')}
                  min={1}
                  max={100000000}
                  suffix={t('次')}
                  onChange={updateField('CriticalRateLimitNum')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='CriticalRateLimitDuration'
                  label={t('关键接口周期')}
                  min={1}
                  max={1200}
                  suffix={t('秒')}
                  onChange={updateField('CriticalRateLimitDuration')}
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field='SearchRateLimitEnable'
                  label={t('启用搜索速率限制')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={updateField('SearchRateLimitEnable')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='SearchRateLimitNum'
                  label={t('搜索每周期次数')}
                  min={1}
                  max={100000000}
                  suffix={t('次')}
                  onChange={updateField('SearchRateLimitNum')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='SearchRateLimitDuration'
                  label={t('搜索限制周期')}
                  min={1}
                  max={1200}
                  suffix={t('秒')}
                  onChange={updateField('SearchRateLimitDuration')}
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存关键接口与搜索限制')}
              </Button>
            </Row>
          </Form.Section>

          <Form.Section text={t('上传、下载与邮箱验证码限制')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='UploadRateLimitNum'
                  label={t('上传每周期次数')}
                  min={1}
                  max={100000000}
                  suffix={t('次')}
                  onChange={updateField('UploadRateLimitNum')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='UploadRateLimitDuration'
                  label={t('上传限制周期')}
                  min={1}
                  max={1200}
                  suffix={t('秒')}
                  onChange={updateField('UploadRateLimitDuration')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='DownloadRateLimitNum'
                  label={t('下载每周期次数')}
                  min={1}
                  max={100000000}
                  suffix={t('次')}
                  onChange={updateField('DownloadRateLimitNum')}
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='DownloadRateLimitDuration'
                  label={t('下载限制周期')}
                  min={1}
                  max={1200}
                  suffix={t('秒')}
                  onChange={updateField('DownloadRateLimitDuration')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='EmailVerificationMaxRequests'
                  label={t('邮箱验证码每周期次数')}
                  min={1}
                  max={100}
                  suffix={t('次')}
                  onChange={updateField('EmailVerificationMaxRequests')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='EmailVerificationDuration'
                  label={t('邮箱验证码限制周期')}
                  min={1}
                  max={1200}
                  suffix={t('秒')}
                  onChange={updateField('EmailVerificationDuration')}
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存上传下载与验证码限制')}
              </Button>
            </Row>
          </Form.Section>

          <Form.Section text={t('模型请求速率限制')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'ModelRequestRateLimitEnabled'}
                  label={t('启用用户模型请求速率限制（可能会影响高并发性能）')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) => {
                    setInputs({
                      ...inputs,
                      ModelRequestRateLimitEnabled: value,
                    });
                  }}
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('限制周期')}
                  step={1}
                  min={0}
                  suffix={t('分钟')}
                  extraText={t('频率限制的周期（分钟）')}
                  field={'ModelRequestRateLimitDurationMinutes'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      ModelRequestRateLimitDurationMinutes: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('用户每周期最多请求次数')}
                  step={1}
                  min={0}
                  max={100000000}
                  suffix={t('次')}
                  extraText={t('包括失败请求的次数，0代表不限制')}
                  field={'ModelRequestRateLimitCount'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      ModelRequestRateLimitCount: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('用户每周期最多请求完成次数')}
                  step={1}
                  min={1}
                  max={100000000}
                  suffix={t('次')}
                  extraText={t('只包括请求成功的次数')}
                  field={'ModelRequestRateLimitSuccessCount'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      ModelRequestRateLimitSuccessCount: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={24} md={16} lg={16} xl={16}>
                <Form.TextArea
                  field='ModelRequestRateLimitGroup'
                  label={t('模型限流分组覆盖')}
                  extraText={t(
                    'JSON 格式：分组名对应 [总请求次数, 成功请求次数]，例如 {"default":[0,1000]}；0 表示不限制总请求数',
                  )}
                  autosize={{ minRows: 3, maxRows: 8 }}
                  onChange={updateField('ModelRequestRateLimitGroup')}
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存模型速率限制')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
