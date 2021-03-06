import moment from 'moment';
import qs from 'query-string';

export function FormatTime(time, formatStr = 'YYYY-MM-DD HH:mm:ss') {
  return moment(time).format(formatStr);
}

export const renderTemplate = data => {
  let domArr = [];
  const render = (object, parent) => {
    _.forEach(object, (v, k) => {
      const name = parent ? `${parent}.${k}` : k;
      if (_.isObject(v) && !_.isArray(v)) {
        render(v, name);
      } else {
        const dom = 'test';
        if (dom) {
          domArr.push(dom);
        }
      }
    });
  };
  render(data);
  return domArr;
};

export const getQuery = str => {
  const query = qs.parse(str);
  return query;
};
