#!/usr/bin/env escript
%% Generates the fixture vectors of `fovea verify` (spec v0.4, 15): fresh,
%% synthetic keys, a realm member endorsement and claim observations signed as
%% macula 13 signs them, and the assessment revision they name as a git bundle.
%% Run with macula's compiled beams on ERL_LIBS and git on PATH:
%%
%%   ERL_LIBS=<_build/default/lib> escript generate.escript <out dir>
%%
%% Every vector but the two genuine ones fails exactly one verification step;
%% its file name says which. No private key is written out.
-mode(compile).

-define(PROFILE, pq_hybrid).
-define(REALM_NAME, <<"fixture.example.org">>).
-define(SYSTEM, <<"fixture-kx">>).
-define(CLAIM, <<"kx_only">>).
-define(ADDRESS, <<"192.0.2.10:4433">>).
-define(STATION, binary:decode_hex(<<"0011111111111111111111111111111111111111111111111111111111111111">>)).
-define(DAY, 86400000).

main([Out]) ->
    ok = filelib:ensure_path(Out),
    {ok, Realm} = macula_node_keys:generate(realm, ?PROFILE),
    {ok, OtherRealm} = macula_node_keys:generate(realm, ?PROFILE),
    {ok, A} = macula_node_keys:generate(identity, ?PROFILE),
    {ok, B} = macula_node_keys:generate(identity, ?PROFILE),
    {ok, NodeA} = macula_node_keys:node_id(A),
    {ok, NodeB} = macula_node_keys:node_id(B),
    RealmId = crypto:hash(sha256, ?REALM_NAME),
    Now = erlang:system_time(millisecond),
    Sha = assessment(Out, NodeA),
    write(Out, "realm_key", macula_node_keys:public_key(Realm)),
    write(Out, "endorsement", endorsement(Realm, RealmId, NodeA, Now - ?DAY, Now + 29 * ?DAY)),
    write(Out, "endorsement_b", endorsement(Realm, RealmId, NodeB, Now - ?DAY, Now + 29 * ?DAY)),
    write(Out, "s3_endorsement_other_signer", endorsement(OtherRealm, RealmId, NodeA, Now - ?DAY, Now + 29 * ?DAY)),
    Elsewhere = crypto:hash(sha256, <<"elsewhere.example.org">>),
    write(Out, "s4_endorsement_other_realm", endorsement(Realm, Elsewhere, NodeA, Now - ?DAY, Now + 29 * ?DAY)),
    write(Out, "s6_endorsement_expired", endorsement(Realm, RealmId, NodeA, Now - 20 * ?DAY, Now - 10 * ?DAY)),
    Holding = payload(RealmId, Sha, Now),
    Accepted = groups(#{<<"secp384r1_mlkem1024">> => 0, <<"secp256r1_mlkem768">> => 0}),
    Refused = groups(#{<<"x25519">> => 1, <<"secp256r1">> => 1, <<"secp384r1">> => 1}),
    BrokenOutcomes = maps:merge(maps:merge(Accepted, Refused), groups(#{<<"x25519">> => 0})),
    write(Out, "obs_holding", observation(Holding, A)),
    write(Out, "obs_broken", observation(Holding#{<<"state">> => 1, <<"outcomes">> => BrokenOutcomes}, A)),
    write(Out, "obs_by_b", observation(Holding, B)),
    write(Out, "s1_type", observation(16#24, Holding, subject(), A)),
    write(Out, "s2_extra_key", observation(Holding#{<<"note">> => {text, <<"x">>}}, A)),
    write(Out, "s2_missing_key", observation(maps:remove(<<"publish">>, Holding), A)),
    write(Out, "s2_state_range", observation(Holding#{<<"state">> => 3}, A)),
    write(Out, "s2_observed_after", observation(Holding#{<<"observed_at">> => Now + ?DAY}, A)),
    write(Out, "s2_subject", observation(16#23, Holding, <<?SYSTEM/binary, 0, ?CLAIM/binary, 0, "192.0.2.11:4433">>, A)),
    write(Out, "s2_text_as_bytes", observation(Holding#{<<"system">> => ?SYSTEM}, A)),
    write(Out, "s8_state_wrong", observation(Holding#{<<"outcomes">> => BrokenOutcomes}, A)),
    write(Out, "s8_outcome_groups", observation(Holding#{<<"outcomes">> => maps:remove({text, <<"secp384r1">>}, maps:merge(Accepted, Refused))}, A)),
    Fewer = maps:merge(Accepted, groups(#{<<"x25519">> => 1})),
    write(Out, "s9_expected_differs", observation(Holding#{<<"expected">> => Fewer, <<"outcomes">> => Fewer}, A)),
    write(Out, "s9_station_node", observation(Holding#{<<"station_node">> => <<16#00, 16#22:248>>}, A)),
    write(Out, "s9_unknown_sha", observation(Holding#{<<"assessment_sha">> => <<16#fe:160>>}, A)),
    write(Out, "s9_other_system", observation(16#23, Holding#{<<"system">> => {text, <<"fixture-other">>}},
                                              <<"fixture-other", 0, ?CLAIM/binary, 0, ?ADDRESS/binary>>, A)),
    ok = file:write_file(filename:join(Out, "assessment_sha"), [binary:encode_hex(Sha, lowercase), $\n]),
    io:format("observer ~s~n", [binary:encode_hex(NodeA, lowercase)]).

endorsement(Key, RealmId, Member, From, Until) ->
    Unsigned = macula_record:realm_member_endorsement(RealmId, #{realm => RealmId, member_node => Member, roles => [<<"peer">>]},
                                                      #{valid_from => From, valid_until => Until, ttl_ms => 30 * ?DAY}),
    macula_record:encode(macula_record:sign(Unsigned, Key)).

subject() -> <<?SYSTEM/binary, 0, ?CLAIM/binary, 0, ?ADDRESS/binary>>.

observation(Payload, Key) -> observation(16#23, Payload, subject(), Key).

observation(Type, Payload, Subject, Key) ->
    Record = macula_record:envelope(Type, maps:from_list([{{text, K}, V} || K := V <- Payload]),
                                    #{subject_id => Subject, ttl_ms => 7 * ?DAY}),
    macula_record:encode(macula_record:sign(Record, Key)).

%% A map of groups, keyed by text as spec 15 writes them.
groups(Values) -> maps:from_list([{{text, G}, V} || G := V <- Values]).

%% Spec 15's payload of a holding round of the fixture's one claim.
payload(RealmId, Sha, Now) ->
    Expected = groups(#{<<"secp384r1_mlkem1024">> => 0, <<"secp256r1_mlkem768">> => 0,
                        <<"x25519">> => 1, <<"secp256r1">> => 1, <<"secp384r1">> => 1}),
    #{<<"system">> => {text, ?SYSTEM}, <<"claim_id">> => {text, ?CLAIM}, <<"realm_id">> => RealmId,
      <<"state">> => 0, <<"probe">> => {text, <<"kx_group">>}, <<"probe_version">> => 1,
      <<"target_address">> => {text, ?ADDRESS}, <<"station_node">> => ?STATION,
      <<"expected">> => Expected, <<"outcomes">> => Expected, <<"assessment_sha">> => Sha,
      <<"publish">> => 0, <<"observed_at">> => Now - 1000}.

%% The assessment revision, committed with a fixed author and date and bundled.
assessment(Out, Observer) ->
    Repo = filename:join(Out, "repo"),
    Dir = filename:join([Repo, "fixture-kx", "cells"]),
    ok = filelib:ensure_path(Dir),
    ok = file:write_file(filename:join([Repo, "fixture-kx", "fovea.yaml"]),
        ["fovea: \"0.4\"\nsystem: fixture-kx\nowner: fixture@example.org\n",
         "attributes:\n  core: [confidentiality, integrity, availability, authenticity, accountability]\n",
         "  enabled: []\n  disabled_justifications:\n    possession: none\n    utility: none\n",
         "columns:\n  actors: [internal, external, trusted_partner, machine_agent]\n",
         "  lifecycle: [create, acquire, deliver, operate, admin, decommission]\n",
         "  data: [at_rest, in_motion, in_use]\n  environment: [physical_natural, socio_legal, temporal]\n",
         "cells_dir: cells/\n",
         "targets:\n  fleet:\n    kind: macula_station\n    stations:\n",
         "      - address: \"192.0.2.10:4433\"\n",
         "        node_id: \"0011111111111111111111111111111111111111111111111111111111111111\"\n",
         "policy:\n  publish: every_result\n  cadence: PT1H\n  suspended: []\n",
         "  observers:\n    - \"", binary:encode_hex(Observer, lowercase), "\"\n"]),
    ok = file:write_file(filename:join(Dir, "in_motion.confidentiality.yaml"),
        ["id: in_motion.confidentiality\nstatus: assessed\nowner: fixture@example.org\n",
         "threat:\n  definition: in_motion threatens confidentiality\n  manifestations:\n    - observed\n",
         "defense:\n  detection:\n    - measure: probed\n      status: by_design\n      source: fixture\n",
         "  countermeasures:\n    - measure: pq only\n      status: by_design\n      source: fixture\n",
         "      evidence:\n        - kind: probe\n          claim: kx_only\n          probe: kx_group\n          version: 1\n",
         "          target: fleet\n          expect:\n",
         "            accepted: [secp384r1_mlkem1024, secp256r1_mlkem768]\n",
         "            refused: [x25519, secp256r1, secp384r1]\n",
         "  recovery: []\n"]),
    Env = "GIT_AUTHOR_NAME=fixture GIT_AUTHOR_EMAIL=fixture@example.org GIT_AUTHOR_DATE=2026-01-01T00:00:00Z "
          "GIT_COMMITTER_NAME=fixture GIT_COMMITTER_EMAIL=fixture@example.org GIT_COMMITTER_DATE=2026-01-01T00:00:00Z ",
    Git = "cd " ++ Repo ++ " && " ++ Env ++ "git -c init.defaultBranch=main ",
    "" = os:cmd(Git ++ "init -q && " ++ Env ++ "git add -A && " ++ Env ++ "git commit -q -m fixture 2>&1"),
    Bundle = filename:absname(filename:join(Out, "assessment.bundle")),
    _ = os:cmd("cd " ++ Repo ++ " && git bundle create -q " ++ Bundle ++ " main 2>&1"),
    Sha = string:trim(os:cmd("cd " ++ Repo ++ " && git rev-parse HEAD")),
    "" = os:cmd("rm -rf " ++ Repo),
    binary:decode_hex(list_to_binary(Sha)).

write(Out, Name, Bytes) ->
    ok = file:write_file(filename:join(Out, Name ++ ".hex"), [binary:encode_hex(Bytes, lowercase), $\n]).
