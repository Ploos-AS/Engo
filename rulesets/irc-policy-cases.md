# IRC policy acceptance cases

These cases define the M2.0 reference policy's expected behavior and are the review contract for deployment.

| Case | Local Engo permission | authenticated/2 | IRC role / explicit grant | BotLogic result |
|---|---|---|---|---|
| unknown account | no | no | none | local deny before BotLogic |
| locally denied account | no | yes | operator | local deny before BotLogic |
| authenticated, no policy grant | yes | yes | none | deny |
| authenticated explicit account grant | yes | yes | account_command/2 | allow |
| authenticated operator | yes | yes | operator_command/1 + channel_operator/2 | allow |
| unauthenticated operator nick | yes/no | no | operator | deny |
| authenticated voiced user | yes | yes | voiced_command/1 + voiced/2 | allow |
| authenticated but offline explicit grant | yes | yes | account_command/2 only | deny |

BotLogic never upgrades Engo's local authorization. These cases intentionally distinguish local authorization from the second policy gate.
