// Shim for antd - project uses semi-ui, not antd.
// @lobehub/icons -> antd-style -> antd is an optional dependency chain.
// Provide minimal stubs to satisfy imports without bundling the full antd package.

const noop = () => {};
const noopComponent = () => null;

export const theme = {
  useToken: () => ({ theme: {}, token: {} }),
  defaultAlgorithm: noop,
  darkAlgorithm: noop,
};

export const ConfigProvider = noopComponent;
export const message = { info: noop, success: noop, warning: noop, error: noop };
export const Modal = { info: noop, success: noop, warning: noop, error: noop, confirm: noop };
export const notification = { info: noop, success: noop, warning: noop, error: noop, open: noop };
export const Grid = { useBreakpoint: () => ({}) };
export const version = '0.0.0-shim';

export default {
  theme,
  ConfigProvider,
  message,
  Modal,
  notification,
  Grid,
  version,
};