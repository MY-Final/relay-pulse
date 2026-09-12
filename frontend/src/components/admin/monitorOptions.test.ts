import { describe, expect, it } from 'vitest';
import { isKnownMonitorService, mergeTemplateNames } from './monitorOptions';

describe('monitorOptions', () => {
  it('只返回当前服务协议的模板，并保留当前旧模板', () => {
    expect(mergeTemplateNames(
      ['cc-opus-arith', 'cx-grok-arith', 'gm-flash-arith'],
      'cx-legacy-template',
      'cx',
    )).toEqual(['cx-grok-arith', 'cx-legacy-template']);
  });

  it('未知历史 service 不过滤模板，便于管理员修复 xai 误填值', () => {
    expect(isKnownMonitorService('xai')).toBe(false);
    expect(mergeTemplateNames(
      ['cc-opus-arith', 'cx-grok-arith'],
      undefined,
      'xai',
    )).toEqual(['cc-opus-arith', 'cx-grok-arith']);
  });
});
