import PropTypes from 'prop-types';
import { Formik } from 'formik';
import { inject, observer } from 'mobx-react';
import { Row, Col } from 'antd';
import { toJS } from 'mobx';
import FormContent from './FormContent';
import { validateForm } from './validate';
import { argumentsParamtersField } from '@/lib/const';
import styles from './template.module.less';

@inject('stageTemplate')
@observer
export default class StageTemplateForm extends React.Component {
  static propTypes = {
    history: PropTypes.object,
    match: PropTypes.object,
    stageTemplate: PropTypes.object,
    initialFormData: PropTypes.object,
    setTouched: PropTypes.func,
    isValid: PropTypes.bool,
    values: PropTypes.object,
  };

  componentDidMount() {
    const {
      match: { params },
      stageTemplate,
    } = this.props;
    const update = !!_.get(params, 'templateName');
    if (update) {
      stageTemplate.getTemplate(params.templateName);
    }
  }

  handleCancle = () => {
    const { history } = this.props;
    history.push('/stageTemplate');
  };

  mapRequestFormToInitForm = data => {
    const alias = _.get(
      data,
      ['metadata', 'annotations', 'cyclone.dev/alias'],
      ''
    );
    const description = _.get(
      data,
      ['metadata', 'annotations', 'cyclone.dev/description'],
      ''
    );
    const spec = this.generateSpecObj(data);
    return {
      metadata: { alias, description },
      spec,
    };
  };

  generateSpecObj = data => {
    const specData = _.get(data, 'spec', {});
    const args = _.get(specData, 'pod.inputs.arguments', []);
    if (args.length > 2) {
      specData.pod.inputs.arguments = args.filter(
        v => v.name === 'image' || v.name === 'cmd'
      );
    }
    let defaultSpec = {
      pod: {
        inputs: {
          arguments: argumentsParamtersField,
          resources: [],
        },
        outputs: {
          resources: [],
        },
        spec: {
          containers: [
            {
              env: [],
              image: '{{ image }}',
              args: ['/bin/sh", "-e", "-c", "{{{ cmd }}}'],
            },
          ],
        },
      },
    };
    return _.assign(defaultSpec, specData);
  };

  initFormValue = () => {
    const templateInfo = toJS(this.props.stageTemplate.template);
    return this.mapRequestFormToInitForm(templateInfo);
  };

  generateData = data => {
    const metadata = {
      annotations: {
        'cyclone.dev/description': _.get(data, 'metadata.description', ''),
        'cyclone.dev/alias': _.get(data, 'metadata.alias', ''),
      },
    };
    return { metadata, spec: data.spec };
  };

  submit = values => {
    const {
      stageTemplate,
      match: { params },
    } = this.props;
    const submitData = this.generateData(values);
    if (_.get(params, 'templateName')) {
      stageTemplate.updateStageTemplate(submitData, params.templateName, () => {
        this.props.history.replace('/stageTemplate');
      });
    } else {
      stageTemplate.createStageTemplate(submitData, () => {
        this.props.history.replace('/stageTemplate');
      });
    }
  };

  componentWillUnmount() {
    this.props.stageTemplate.resetTemplate();
  }

  render() {
    const {
      match: { params },
    } = this.props;
    const update = !!_.get(params, 'templateName');
    return (
      <div className={styles['stagetemplate-form']}>
        <div className="head-bar">
          <h2>
            {update ? intl.get('template.update') : intl.get('template.create')}
          </h2>
        </div>
        <Row>
          <Col span={20}>
            <Formik
              initialValues={this.initFormValue()}
              enableReinitialize={true}
              validate={validateForm}
              onSubmit={this.submit}
              render={props => (
                <FormContent
                  {...props}
                  update={update}
                  handleCancle={this.handleCancle}
                />
              )}
            />
          </Col>
        </Row>
      </div>
    );
  }
}
