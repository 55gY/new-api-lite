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

import { API } from '../helpers';

export class SecureVerificationService {
  static async checkAvailableVerificationMethods() {
    try {
      const twoFAResponse = await API.get('/api/user/2fa/status');
      const has2FA =
        twoFAResponse.data?.success &&
        twoFAResponse.data?.data?.enabled === true;
      return { has2FA };
    } catch (error) {
      console.error('Failed to check verification methods:', error);
      return { has2FA: false };
    }
  }

  static async verify2FA(code) {
    if (!code?.trim()) {
      throw new Error('请输入验证码或备用码');
    }

    const verifyResponse = await API.post('/api/verify', {
      method: '2fa',
      code: code.trim(),
    });

    if (!verifyResponse.data?.success) {
      throw new Error(verifyResponse.data?.message || '验证失败');
    }
  }

  static async verify(method, code = '') {
    if (method !== '2fa') {
      throw new Error(`不支持的验证方式: ${method}`);
    }
    return await this.verify2FA(code);
  }
}

export const createApiCalls = {
  custom:
    (url, method = 'POST', extraData = {}) =>
    async () => {
      const data = extraData;

      let response;
      switch (method.toUpperCase()) {
        case 'GET':
          response = await API.get(url, { params: data });
          break;
        case 'POST':
          response = await API.post(url, data);
          break;
        case 'PUT':
          response = await API.put(url, data);
          break;
        case 'DELETE':
          response = await API.delete(url, { data });
          break;
        default:
          throw new Error(`不支持的HTTP方法: ${method}`);
      }
      return response.data;
    },
};
