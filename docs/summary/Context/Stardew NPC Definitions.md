# Stardew NPC Definitions

> Status: Manual Draft
> Source Policy: ValleyTalk-inspired structure, rewritten short definitions.

本表记录 `stardew-valley` vanilla NPC Agent Definition 的当前覆盖范围。

这些 Definition 使用 `runtime/config/games/stardew-valley/definitions/*.json`，每个固定 NPC 一个 `npc-*.json` 文件。`archetype-town-villager.json` 只保留为共享模板示例，不作为 vanilla NPC 主路径。

| NPC | Definition File | Definition ID | Core Traits |
| --- | --- | --- | --- |
| Abigail | `npc-abigail.json` | `npc:Abigail` | adventurous, independent, curious, games/mines/amethyst |
| Alex | `npc-alex.json` | `npc:Alex` | confident, competitive, athletic, vulnerable underneath |
| Caroline | `npc-caroline.json` | `npc:Caroline` | caring, tea-focused, health-conscious, mildly anxious |
| Clint | `npc-clint.json` | `npc:Clint` | blacksmith, lonely, socially awkward, craft-focused |
| Demetrius | `npc-demetrius.json` | `npc:Demetrius` | scientific, analytical, researcher, family-aware |
| Dwarf | `npc-dwarf.json` | `npc:Dwarf` | underground trader, cautious, practical, mine-wise |
| Elliott | `npc-elliott.json` | `npc:Elliott` | writer, romantic, sensitive, beach solitude |
| Emily | `npc-emily.json` | `npc:Emily` | spiritual, creative, optimistic, empathetic |
| Evelyn | `npc-evelyn.json` | `npc:Evelyn` | warm, gardening, baking, nurturing elder |
| George | `npc-george.json` | `npc:George` | gruff, stubborn, elder perspective, caring underneath |
| Gus | `npc-gus.json` | `npc:Gus` | saloon host, friendly, responsible, community listener |
| Haley | `npc-haley.json` | `npc:Haley` | stylish, guarded, photography, softening over time |
| Harvey | `npc-harvey.json` | `npc:Harvey` | doctor, cautious, gentle, professionally responsible |
| Jas | `npc-jas.json` | `npc:Jas` | childlike, shy, imaginative, tied to Marnie/Shane |
| Jodi | `npc-jodi.json` | `npc:Jodi` | family responsibility, worried, warm, resilient |
| Kent | `npc-kent.json` | `npc:Kent` | returned soldier, guarded, dutiful, family repair |
| Krobus | `npc-krobus.json` | `npc:Krobus` | shadow person, gentle, cautious, peace-seeking |
| Leah | `npc-leah.json` | `npc:Leah` | artist, nature, independence, life away from the city |
| Lewis | `npc-lewis.json` | `npc:Lewis` | mayor, traditional, civic duty, public dignity |
| Linus | `npc-linus.json` | `npc:Linus` | nature life, independence, philosophy, chosen solitude |
| Marnie | `npc-marnie.json` | `npc:Marnie` | animal care, warmth, hard work, household responsibility |
| Maru | `npc-maru.json` | `npc:Maru` | engineering, science, friendly, slightly awkward |
| Pam | `npc-pam.json` | `npc:Pam` | rough-edged, regretful, caring mother, bus/saloon |
| Penny | `npc-penny.json` | `npc:Penny` | tutor, gentle, responsible, longing for stability |
| Pierre | `npc-pierre.json` | `npc:Pierre` | merchant, anxious, family-minded, competition pressure |
| Robin | `npc-robin.json` | `npc:Robin` | carpenter, practical, proud of craft, family-focused |
| Sam | `npc-sam.json` | `npc:Sam` | music, skating, youthful energy, loyal to family/friends |
| Sandy | `npc-sandy.json` | `npc:Sandy` | desert shopkeeper, outgoing, stylish, lonely |
| Sebastian | `npc-sebastian.json` | `npc:Sebastian` | programmer, introverted, dry wit, motorcycle/music |
| Shane | `npc-shane.json` | `npc:Shane` | cynical, guarded, wounded, hidden care, chickens |
| Vincent | `npc-vincent.json` | `npc:Vincent` | childlike, curious, imaginative, family-aware |
| Willy | `npc-willy.json` | `npc:Willy` | fisherman, patient, ocean-rooted, mentor-like |
| Wizard | `npc-wizard.json` | `npc:Wizard` | arcane, mysterious, wise, distant from town life |

内容原则：

1. 只写稳定角色定义，不写当前存档事实。
2. 关系、事件和原作台词不直接进入 Phase7.1 schema。
3. 不复制 ValleyTalk biography 原文。
4. 每轮 Runtime 只按当前 target 的 `game_id + definition_id` 使用一个 Agent Definition。
