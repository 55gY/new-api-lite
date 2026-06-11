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

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import zhCNTranslation from './locales/zh-CN.json';

i18n
  .use(initReactI18next)
  .init({
    load: 'currentOnly',
    supportedLngs: ['zh-CN'],
    resources: {
      'zh-CN': zhCNTranslation,
    },
    fallbackLng: 'zh-CN',
    lng: 'zh-CN',
    nsSeparator: false,
    interpolation: {
      escapeValue: false,
    },
  });

window.__i18n = i18n;
localStorage.setItem('i18nextLng', 'zh-CN');

export default i18n;
