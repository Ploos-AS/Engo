% Engo M2.0 reference BotLogic policy.
%
% This ruleset is deliberately restrictive. Engo's local account permission
% check remains the primary authority; may_execute/2 is only a second gate.
%
% Operators may execute commands explicitly declared operator_command/1.
% Voiced users may execute commands explicitly declared voiced_command/1.
% Account-specific grants remain explicit and require authenticated/2.
%
% Configure the command classes below for the deployment. Empty classes mean
% no channel-role grants.

% Example declarations (disabled):
% operator_command(reload).
% voiced_command(topic).
% account_command(admin_account, reload).

may_execute(Account, Command) :-
    authenticated(Nick, Account),
    account_command(Account, Command),
    online(Nick).

may_execute(Account, Command) :-
    authenticated(Nick, Account),
    channel_operator(_, Nick),
    operator_command(Command).

may_execute(Account, Command) :-
    authenticated(Nick, Account),
    voiced(_, Nick),
    voiced_command(Command).

% Empty-by-default extension points keep the reference policy deny-by-default.
account_command('__engo_none__', '__engo_none__').
operator_command('__engo_none__').
voiced_command('__engo_none__').
