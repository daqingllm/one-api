import React, { useEffect, useState } from 'react';
import { Button, Form, Header, Label, Pagination, Segment, Icon, Table } from 'semantic-ui-react';
import { API, isAdmin, showError, timestamp2string } from '../helpers';

import { ITEMS_PER_PAGE } from '../constants';
import { renderQuota } from '../helpers/render';

const adminColumns = [
  {
    title: '渠道',
    dataIndex: 'channel',
    width: 1,
    label: true,
    basic: true,
  },
  {
    title: '用户',
    dataIndex: 'username',
    width: 1,
    label: true,
    basic: false,
  },
  {
    title: '详情',
    dataIndex: 'content',
    width: 4,
  },
  {
    title: '类型',
    dataIndex: 'type',
    width: 1,
  }
]

const defaultColumns = [
  {
    title: 'key',
    dataIndex: 'token_name',
    width: 1,
    label: true,
    basic: true,
  },
  {
    title: '模型',
    dataIndex: 'model_name',
    width: 2,
    label: true,
    basic: true,
  },
  {
    title: '输入',
    dataIndex: 'prompt_tokens',
    width: 1,
  },
  {
    title: '输出',
    dataIndex: 'completion_tokens',
    width: 1,
  },
  {
    title: '额度',
    dataIndex: 'quota',
    width: 1,
  },
  {
    title: '时间',
    dataIndex: 'created_at',
    width: 3,
  },
]

const specialField = ['type', 'quota'];

const MODE_OPTIONS = [
  { key: 'all', text: '全部用户', value: 'all' },
  { key: 'self', text: '当前用户', value: 'self' }
];

const LOG_OPTIONS = [
  { key: '0', text: '全部', value: 0 },
  { key: '1', text: '充值', value: 1 },
  { key: '2', text: '消费', value: 2 },
  { key: '3', text: '管理', value: 3 },
  { key: '4', text: '系统', value: 4 }
];

function renderType(type) {
  switch (type) {
    case 1:
      return <Label basic color='green'> 充值 </Label>;
    case 2:
      return <Label basic color='olive'> 消费 </Label>;
    case 3:
      return <Label basic color='orange'> 管理 </Label>;
    case 4:
      return <Label basic color='purple'> 系统 </Label>;
    default:
      return <Label basic color='black'> 未知 </Label>;
  }
}

const LogsTable = () => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [downloadLoading, setDownloadLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const isAdminUser = isAdmin();
  let now = new Date();
  const [inputs, setInputs] = useState({
    username: '',
    token_name: '',
    model_name: '',
    start_timestamp: timestamp2string(0),
    end_timestamp: timestamp2string(now.getTime() / 1000 + 3600),
    channel: '',
    logType: 0
  });
  const columns = [...(isAdminUser? adminColumns : []), ...defaultColumns];
  const { username, token_name, logType, model_name, start_timestamp, end_timestamp, channel } = inputs;

  const [stat, setStat] = useState({
    quota: 0,
    token: 0
  });

  const handleInputChange = (e, { name, value }) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const getLogSelfStat = async () => {
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let res = await API.get(`/api/log/self/stat?type=${logType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const getLogStat = async () => {
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let res = await API.get(`/api/log/stat?type=${logType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}`);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const handleLogStat = async () => {
      if (isAdminUser) {
        await getLogStat();
      } else {
        await getLogSelfStat();
      }
  };

  const loadLogs = async (startIdx) => {
    let url = '';
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    if (isAdminUser) {
      url = `/api/log/?p=${startIdx}&type=${logType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}`;
    } else {
      url = `/api/log/self/?p=${startIdx}&type=${logType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`;
    }
    const res = await API.get(url);
    const { success, message, data } = res.data;
    const curData = data.map(item => ({
      ...item,
      created_at: timestamp2string(item.created_at)
    }))
    if (success) {
      if (startIdx === 0) {
        setLogs(curData);
      } else {
        let newLogs = [...logs];
        newLogs.splice(startIdx * ITEMS_PER_PAGE, curData.length, ...curData);
        setLogs(newLogs);
      }
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const onPaginationChange = (e, { activePage }) => {
    (async () => {
      if (activePage === Math.ceil(logs.length / ITEMS_PER_PAGE) + 1) {
        // In this case we have to load more data and then append them.
        await loadLogs(activePage - 1);
      }
      setActivePage(activePage);
    })();
  };

  const refresh = async () => {
    setLoading(true);
    setActivePage(1);
    // handleLogStat()
    await loadLogs(0);
  };

  const sortLog = (key) => {
    if (logs.length === 0) return;
    setLoading(true);
    let sortedLogs = [...logs];
    if (typeof sortedLogs[0][key] === 'string') {
      sortedLogs.sort((a, b) => {
        return ('' + a[key]).localeCompare(b[key]);
      });
    } else {
      sortedLogs.sort((a, b) => {
        if (a[key] === b[key]) return 0;
        if (a[key] > b[key]) return -1;
        if (a[key] < b[key]) return 1;
      });
    }
    if (sortedLogs[0].id === logs[0].id) {
      sortedLogs.reverse();
    }
    setLogs(sortedLogs);
    setLoading(false);
  };
  // 批量下载
  const batchExport = async () => {
    setDownloadLoading(true)
    let url = '';
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    if (isAdminUser) {
      url = `/api/log/?p=0&num=10000&type=${logType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}`;
    } else {
      url = `/api/log/self/?p=&num=10000&type=${logType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`;
    }
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      const curData = data.map(item => ({
        ...item,
        created_at: timestamp2string(item.created_at)
      }))
      // 下载
      const tableRows = [
        columns.map(item => item.title), // 第一行就是表格表头
        ...curData.map(log => columns.map(item => log[item.dataIndex]))
      ]
      // 构造数据字符，换行需要用\r\n
      let CsvString = tableRows.map(data => data.join(',')).join('\r\n');
      // 加上 CSV 文件头标识
      CsvString = 'data:application/vnd.ms-excel;charset=utf-8,\uFEFF' + encodeURIComponent(CsvString);
      // 通过创建a标签下载
      const link = document.createElement('a');
      link.href = CsvString;
      // 对下载的文件命名
      link.download = `使用明细.csv`;
      // 模拟点击下载
      link.click();
      // 移除a标签
      link.remove();
      setDownloadLoading(false)
    } else {
      showError(message);
      setDownloadLoading(false)
    }
  }

  useEffect(() => {
    refresh().then();
  }, []);

  return (
    <>
      <Segment>
        <Form>
          <Form.Group>
            <Form.Input fluid label={'Key名称'} width={3} value={token_name}
                        placeholder={'可选值'} name='token_name' onChange={handleInputChange} />
            <Form.Input fluid label='模型名称' width={3} value={model_name} placeholder='可选值'
                        name='model_name'
                        onChange={handleInputChange} />
            <Form.Select fluid label='明细分类' width={2} options={LOG_OPTIONS} value={logType} onChange={(e, o) => handleInputChange(e, { name: 'logType', value: o.value })}/>
            <Form.Input fluid label='起始时间' width={4} value={start_timestamp} type='datetime-local'
                        name='start_timestamp'
                        onChange={handleInputChange} />
            <Form.Input fluid label='结束时间' width={4} value={end_timestamp} type='datetime-local'
                        name='end_timestamp'
                      onChange={handleInputChange} />
          </Form.Group>
          <Form.Group>
            {
            isAdminUser && <>
                <Form.Input fluid label={'渠道 ID'} width={3} value={channel}
                            placeholder='可选值' name='channel'
                            onChange={handleInputChange} />
                <Form.Input fluid label={'用户名称'} width={3} value={username}
                            placeholder={'可选值'} name='username'
                            onChange={handleInputChange} />
            </>
            }
            <Form.Field inline width={isAdminUser ? 8 : 14} />
            <Form.Button fluid label={isAdminUser ? '操作' : ''} width={2} onClick={refresh} loading={loading}>查询</Form.Button>
          </Form.Group>
        
        </Form>
        <Segment clearing textAlign='left' basic style={{ padding: 0 }}>
          <Header as='h4' floated='left' style={{ marginBottom: 0, lineHeight: '33px' }}>
            使用明细
          </Header>
          <Button as='div' labelPosition='left' floated='right' onClick={batchExport} loading={downloadLoading}>
            <Label as='span' basic>
              最多10000条
            </Label>
            <Button icon>
              <Icon name='download' />
            </Button>
          </Button>
        </Segment>
        <Table basic compact size='small'>
          <Table.Header>
            <Table.Row>
              {columns.map(item => 
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog(item.dataIndex);
                  }}
                  width={item.width}
                  key={item.dataIndex}
                >
                  {item.title}
                </Table.HeaderCell>
                )}
            </Table.Row>
          </Table.Header>

          <Table.Body>
            {logs
              .slice(
                (activePage - 1) * ITEMS_PER_PAGE,
                activePage * ITEMS_PER_PAGE
              )
              .map((log, idx) => {
                if (log.deleted) return <></>;
                return (
                  <Table.Row key={log.id}>
                    {columns.map(item => 
                      <Table.Cell>
                        {item.dataIndex === 'type' ? renderType(log.type) : item.dataIndex === 'quota' ? renderQuota(log.quota, 6) : ''}
                        {specialField.includes(item.dataIndex) ? '' : item.label? <Label basic={item.basic}>{log[item.dataIndex]}</Label> : log[item.dataIndex]}
                      </Table.Cell>)}
                  </Table.Row>
                );
              })}
          </Table.Body>

          <Table.Footer>
            <Table.Row>
              <Table.HeaderCell colSpan={'10'}>
                <Pagination
                  floated='right'
                  activePage={activePage}
                  onPageChange={onPaginationChange}
                  size='small'
                  siblingRange={1}
                  totalPages={
                    Math.ceil(logs.length / ITEMS_PER_PAGE) +
                    (logs.length % ITEMS_PER_PAGE === 0 ? 1 : 0)
                  }
                />
              </Table.HeaderCell>
            </Table.Row>
          </Table.Footer>
        </Table>
      </Segment>
    </>
  );
};

export default LogsTable;
