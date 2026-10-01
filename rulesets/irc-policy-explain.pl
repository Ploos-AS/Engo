% Companion explainable policy for BotLogic /v1/explain.
% Keep these clauses aligned with rulesets/irc-policy.pl.

may_execute_explain(Account, Command, Proof) :-
    authenticated(Nick, Account),
    account_command(Account, Command),
    online(Nick),
    Proof = 'authenticated-account-explicit-command'.

may_execute_explain(Account, Command, Proof) :-
    authenticated(Nick, Account),
    channel_operator(Channel, Nick),
    operator_command(Command),
    Channel = Channel,
    Proof = 'authenticated-channel-operator'.

may_execute_explain(Account, Command, Proof) :-
    authenticated(Nick, Account),
    voiced(Channel, Nick),
    voiced_command(Command),
    Channel = Channel,
    Proof = 'authenticated-voiced-user'.
