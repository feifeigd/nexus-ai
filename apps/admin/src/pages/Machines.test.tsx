import { Form } from 'antd';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import {
  MachineClearPrivateKeyField,
  MachineTable,
  type Machine,
} from './Machines';

const bareMetal: Machine = {
  id: 'machine_1',
  name: 'web-1',
  privateAddress: '10.0.0.1',
  publicAddress: '1.2.3.4',
  region: 'cn-east',
  rootPublicKey: 'ssh-rsa AAAA',
  joinedK8s: false,
  createdAt: '2026-11-06T02:00:00Z',
  updatedAt: '2026-11-06T02:00:00Z',
};

describe('MachineTable', () => {
  it('renders machine fields and edit/delete/exec actions', () => {
    const html = renderToStaticMarkup(
      <MachineTable
        machines={[bareMetal]}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onExec={vi.fn()}
      />,
    );

    expect(html).toContain('web-1');
    expect(html).toContain('10.0.0.1');
    expect(html).toContain('1.2.3.4');
    expect(html).toContain('cn-east');
    expect(html).toContain('ssh-rsa AAAA');
    expect(html).toContain('未加入');
    expect(html).toContain('执行命令');
    expect(html).toContain('编辑');
    expect(html).toContain('删除');
  });

  it('shows the joined-k8s tag and omits optional fields with a dash', () => {
    const html = renderToStaticMarkup(
      <MachineTable
        machines={[
          { ...bareMetal, publicAddress: undefined, region: undefined, joinedK8s: true },
        ]}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onExec={vi.fn()}
      />,
    );

    expect(html).toContain('已加入');
    expect(html).toContain('—');
  });
});

describe('MachineClearPrivateKeyField', () => {
  it('renders an explicit clear-key switch with a non-destructive machine warning', () => {
    const html = renderToStaticMarkup(
      <Form>
        <MachineClearPrivateKeyField />
      </Form>,
    );

    expect(html).toContain('清除已保存的 SSH 私钥');
    expect(html).toContain('开启后保存会清除私钥，但保留机器');
    expect(html).toContain('清除');
    expect(html).toContain('保留');
  });
});
