/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 * 
 *     http://www.apache.org/licenses/LICENSE-2.0
 * 
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { readFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const pluginRoot = join(__dirname, '..', '..');

const skillNames = ['scaffolding', 'extensions', 'java-interop', 'debug', 'guide', 'migrate', 'development'];

export const skills = skillNames.map(name => {
  const skillPath = join(pluginRoot, 'skills', name, 'SKILL.md');
  let content;
  try {
    content = readFileSync(skillPath, 'utf-8');
  } catch (err) {
    throw new Error(`dubbo-go-agent-skills: failed to read ${skillPath}: ${err.message}`);
  }
  // Parse frontmatter (tolerates both LF and CRLF line endings)
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/);
  const frontmatter = match ? match[1] : '';
  const body = match ? match[2] : content;
  const nameMatch = frontmatter.match(/^name:\s*(.+)$/m);
  const descMatch = frontmatter.match(/^description:\s*(.+)$/m);
  return {
    name: nameMatch ? nameMatch[1].trim() : `dubbo-go:${name}`,
    description: descMatch ? descMatch[1].trim() : '',
    content: body,
  };
});
