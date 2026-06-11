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

import { useMemo } from 'react';

export const useNavigation = (t, headerNavModules) => {
  const mainNavLinks = useMemo(() => {
    // 默认配置，如果没有传入配置则显示所有模块
    const defaultModules = {
      home: true,
      models: true,
      console: true,
    };

    // 使用传入配置覆盖默认配置，避免旧配置缺少新增模块时被隐藏
    const modules = { ...defaultModules, ...(headerNavModules || {}) };

    const allLinks = [
      {
        text: t('首页'),
        itemKey: 'home',
        to: '/',
      },
      {
        text: t('模型列表'),
        itemKey: 'models',
        to: '/models',
      },
      {
        text: t('控制台'),
        itemKey: 'console',
        to: '/console',
      },
    ];

    // 根据配置过滤导航链接
    return allLinks.filter((link) => modules[link.itemKey] === true);
  }, [t, headerNavModules]);

  return {
    mainNavLinks,
  };
};
