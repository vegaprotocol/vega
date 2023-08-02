Feature: Simple Spot Order between two parties match successfully
  Scenario: Simple Spot Order matches with counter party
  Background:
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | default-none     | default-basic |


    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount                       |
      | 92bf2b120913090e5f09fa92e96f972a4d325c84fa662ed52ae1aba81e1ba785 | ETH   | 1000000000000000013287555072 |
      | 92bf2b120913090e5f09fa92e96f972a4d325c84fa662ed52ae1aba81e1ba785 | BTC   | 1000000000000000013287555072 |
      | fac43f8b56111cac2e135b09122af24fc38dac9321bbc884c9e0f945c351357f | ETH   | 1000000000000000013287555072 |
      | fac43f8b56111cac2e135b09122af24fc38dac9321bbc884c9e0f945c351357f | BTC   | 1000000000000000013287555072 |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | ETH   | 1000000000000000013287555072 |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC   | 1000000000000000013287555072 |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | ETH   | 1000000000000000013287555072 |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC   | 1000000000000000013287555072 |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | ETH   | 1000000000000000013287555072 |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC   | 1000000000000000013287555072 |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | ETH   | 1000000000000000013287555072 |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC   | 1000000000000000013287555072 |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | ETH   | 1000000000000000013287555072 |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC   | 1000000000000000013287555072 |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | ETH   | 1000000000000000013287555072 |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC   | 1000000000000000013287555072 |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | ETH   | 1000000000000000013287555072 |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC   | 1000000000000000013287555072 |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | ETH   | 1000000000000000013287555072 |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC   | 1000000000000000013287555072 |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | ETH   | 1000000000000000013287555072 |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC   | 1000000000000000013287555072 |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | ETH   | 1000000000000000013287555072 |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC   | 1000000000000000013287555072 |
      | 441abfca5144ca3aab62f561e3bbdafe568422d9262403f24cc582ba98b7ab6d | ETH   | 1000000000000000013287555072 |
      | 441abfca5144ca3aab62f561e3bbdafe568422d9262403f24cc582ba98b7ab6d | BTC   | 1000000000000000013287555072 |
      | 9f8ad7f8fde333f319741958b8dc87904015a957cd2ac06c3e186257b4ba8866 | ETH   | 1000000000000000013287555072 |
      | 9f8ad7f8fde333f319741958b8dc87904015a957cd2ac06c3e186257b4ba8866 | BTC   | 1000000000000000013287555072 |
      | cdf4eb76dcb48fe5b8d982f7c43ab5920286958a1db29f635df02b98be094849 | ETH   | 1000000000000000013287555072 |
      | cdf4eb76dcb48fe5b8d982f7c43ab5920286958a1db29f635df02b98be094849 | BTC   | 1000000000000000013287555072 |
      | dd4b578851f2feb5fa651e79b2bc4ee857b6c085b66e50db32aa1fdaa590018f | ETH   | 1000000000000000013287555072 |
      | dd4b578851f2feb5fa651e79b2bc4ee857b6c085b66e50db32aa1fdaa590018f | BTC   | 1000000000000000013287555072 |
      | 646292b6ae7dab466a96b6d9be08a2cb5df6370c133b7c845c88b22c71c91327 | ETH   | 1000000000000000013287555072 |
      | 646292b6ae7dab466a96b6d9be08a2cb5df6370c133b7c845c88b22c71c91327 | BTC   | 1000000000000000013287555072 |

    # place orders and generate trades
    And the parties place the following orders:
      | party                                                            | market id | side | volume | price | resulting trades | type       | tif     | reference                             | expires in |
      | 92bf2b120913090e5f09fa92e96f972a4d325c84fa662ed52ae1aba81e1ba785 | BTC/ETH   | buy  | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTT | 669cb1db-a57c-4028-9013-9c1d08746d85  | 120        |
      | fac43f8b56111cac2e135b09122af24fc38dac9321bbc884c9e0f945c351357f | BTC/ETH   | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTT | 3e027d3d-c719-4a87-944b-1877738c4050  | 120        |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9636  | 0                | TYPE_LIMIT | TIF_GTC | cbbc6b1e-373f-428c-b6f2-2bf9bcc04399  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9491  | 0                | TYPE_LIMIT | TIF_GTC | 58a8f2bd-a93b-44a0-a8e7-47318bd4bc10  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9191  | 0                | TYPE_LIMIT | TIF_GTC | 2f93451c-78d5-4fe6-8034-f2da8076cb7b  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9206  | 0                | TYPE_LIMIT | TIF_GTC | eaf0fedd-267b-4cdb-955f-f5553afccaaa  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 10240 | 0                | TYPE_LIMIT | TIF_GTC | 4cba3a27-874e-4e43-ad54-32aabfc1fd76  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9204  | 0                | TYPE_LIMIT | TIF_GTC | af7f6d94-0287-428c-bcc6-8831b133e966  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9393  | 0                | TYPE_LIMIT | TIF_GTC | d4f6b8ab-52cb-44aa-8e36-8df87193ec57  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9870  | 0                | TYPE_LIMIT | TIF_GTC | 6e7c2a7f-0cda-47c1-973a-e37d0affda98  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9462  | 0                | TYPE_LIMIT | TIF_GTC | 33d158f4-9a30-47d7-a1ac-4c7256934e10  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9518  | 0                | TYPE_LIMIT | TIF_GTC | 6cfaed24-9e45-4531-95fc-eba04b39e83f  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10173 | 0                | TYPE_LIMIT | TIF_GTC | c4c63ab9-ab3c-4325-b400-844b01bdf0a3  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10152 | 0                | TYPE_LIMIT | TIF_GTC | 7a555d7a-72f0-4eee-bb13-d5a99f4a0e11  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10353 | 0                | TYPE_LIMIT | TIF_GTC | ac20c37d-1957-4f80-874f-fd43bf781873  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10308 | 0                | TYPE_LIMIT | TIF_GTC | 45c783a2-42b7-4f07-9018-46ef51cb0bd7  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10268 | 0                | TYPE_LIMIT | TIF_GTC | cda04494-dc64-4b62-bcaa-548cd2273627  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10156 | 0                | TYPE_LIMIT | TIF_GTC | 31be1169-44a4-4a9c-927e-fa3c2055b18e  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10209 | 0                | TYPE_LIMIT | TIF_GTC | 5d2042cc-536e-4310-abf6-849fc0d0e28a" |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10227 | 0                | TYPE_LIMIT | TIF_GTC | d8dda3e6-5809-43c9-ad73-a24ba7370ac4  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10726 | 0                | TYPE_LIMIT | TIF_GTC | f30bbb73-3aeb-4af1-889b-05b1ad02323b  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10516 | 0                | TYPE_LIMIT | TIF_GTC | 19876814-4fe2-49fb-ba5f-bf9501c71f29  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9566  | 0                | TYPE_LIMIT | TIF_GTC | 3430bddc-17e8-4f30-936e-8f2cd1c59a9d  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9743  | 0                | TYPE_LIMIT | TIF_GTC | 6c3b8370-04b3-4a52-be0d-f362a4fe79ba  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9668  | 0                | TYPE_LIMIT | TIF_GTC | 92cdc99c-2b6d-44ef-b9c9-c9ecce07ede0  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9483  | 0                | TYPE_LIMIT | TIF_GTC | d1f78e75-6d31-47c6-a2c4-fa3739575cdf  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9627  | 0                | TYPE_LIMIT | TIF_GTC | 6fe9d0a9-4ce2-440f-9355-183cf5406cfe  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9847  | 0                | TYPE_LIMIT | TIF_GTC | ffb2f4c3-20a1-4cf3-a709-acbafeef481c  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9399  | 0                | TYPE_LIMIT | TIF_GTC | 5a2403f0-85ca-4280-8b23-8f025763fcc8  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9857  | 0                | TYPE_LIMIT | TIF_GTC | 8506f460-ca43-4b75-be6c-ef2dbb2cf81f  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9851  | 0                | TYPE_LIMIT | TIF_GTC | 5de81df7-45d9-4b3b-bafd-87b3e5d98c33  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9739  | 0                | TYPE_LIMIT | TIF_GTC | f44896b8-0ffc-4cd8-9fc5-94abd2743fc6  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10300 | 0                | TYPE_LIMIT | TIF_GTC | 9b12d882-e95b-4081-b19e-935aa7801679  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10090 | 0                | TYPE_LIMIT | TIF_GTC | 0469ce7a-d7dc-4702-9125-d5f98dd7e2f9  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10330 | 0                | TYPE_LIMIT | TIF_GTC | b277753c-1539-4527-b0f9-baea94eab8fe  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10235 | 0                | TYPE_LIMIT | TIF_GTC | f2b2a72a-12ba-46b9-b415-5ca6e7edbd31  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10108 | 0                | TYPE_LIMIT | TIF_GTC | 39ae8d1e-42a7-4d42-8e93-6162e2c57bb6  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10263 | 0                | TYPE_LIMIT | TIF_GTC | 095f1793-a410-4e66-bff4-0119da700277  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10536 | 0                | TYPE_LIMIT | TIF_GTC | fc3f3a99-d3ee-40e7-aa56-36932a023ecc  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10266 | 0                | TYPE_LIMIT | TIF_GTC | 03469f95-1861-4b4f-b8cf-0e22d8661be3  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 9911  | 0                | TYPE_LIMIT | TIF_GTC | b7e2b92d-4575-4c51-9e62-066327e8f028  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10313 | 0                | TYPE_LIMIT | TIF_GTC | 7de310bc-1efc-45a8-9f35-5d99d90eee2a  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9798  | 0                | TYPE_LIMIT | TIF_GTC | 35924cc8-35c1-4bff-b80b-448142e8851e  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9735  | 0                | TYPE_LIMIT | TIF_GTC | 59d1abb5-beb7-42c1-940a-cd3e54b5845c  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9642  | 0                | TYPE_LIMIT | TIF_GTC | 9e5afc4c-f63b-4c17-8cd8-5a11aead5a6d  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9724  | 0                | TYPE_LIMIT | TIF_GTC | 4dd0675f-17d4-4632-ae2c-fac849bbe983  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9561  | 0                | TYPE_LIMIT | TIF_GTC | a9e04342-43c0-404b-b72d-a9fc3e318750  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9964  | 0                | TYPE_LIMIT | TIF_GTC | 1181c3eb-422c-4619-8701-ca8b2335c271  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9489  | 0                | TYPE_LIMIT | TIF_GTC | 87de905b-8407-46e1-85d6-437676cbfbe5  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9870  | 0                | TYPE_LIMIT | TIF_GTC | 152f4c97-1208-40b1-a0a5-e9e0323f2f25  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9730  | 0                | TYPE_LIMIT | TIF_GTC | 827ac3d9-26fc-42a6-bf4b-750d02339594  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9498  | 0                | TYPE_LIMIT | TIF_GTC | 6cbcd253-f05f-4187-bed2-630148586d83  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10401 | 0                | TYPE_LIMIT | TIF_GTC | de532646-44ff-4ccd-84a7-35f91c2647fb  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10387 | 0                | TYPE_LIMIT | TIF_GTC | ffb90be7-3b86-4ebf-8a36-6154a8228f4b  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10434 | 0                | TYPE_LIMIT | TIF_GTC | 982a87d4-4cf6-4780-a06d-9804cba68ea7  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10224 | 0                | TYPE_LIMIT | TIF_GTC | a11a1e61-8c52-4bcc-844c-c55b3d7de0d4  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10173 | 0                | TYPE_LIMIT | TIF_GTC | b02ff0c4-9053-409f-8179-686f6d01f213  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10729 | 0                | TYPE_LIMIT | TIF_GTC | 78d9c1fd-9825-460e-95e2-4d290faf3697  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10380 | 0                | TYPE_LIMIT | TIF_GTC | b61dec97-022c-41ef-b9c9-3fd65f4cc7f7  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10207 | 0                | TYPE_LIMIT | TIF_GTC | b396ff5f-ba98-4de7-8b90-0557a428bf34  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10297 | 0                | TYPE_LIMIT | TIF_GTC | a8fc4a09-73f4-4ec5-8fb5-1262226afb93  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10303 | 0                | TYPE_LIMIT | TIF_GTC | 5d3a8e4b-56ea-43c3-abd3-0d8414a9d25d  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9431  | 0                | TYPE_LIMIT | TIF_GTC | 19ea1844-5d99-4783-8f57-c2e5900ef2f3  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9817  | 0                | TYPE_LIMIT | TIF_GTC | d1035cf5-ae0f-45dd-998e-69a7b419d620  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9581  | 0                | TYPE_LIMIT | TIF_GTC | 2eebc6f9-b5ef-4d28-9332-815c64715841  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9669  | 0                | TYPE_LIMIT | TIF_GTC | 724f9ba4-b478-433f-8241-ca7b4922d4fe  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9552  | 0                | TYPE_LIMIT | TIF_GTC | 80d72c81-f50b-4d58-8884-109c5342123e  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9348  | 0                | TYPE_LIMIT | TIF_GTC | d5642957-c792-4043-a091-fb9fa0a2c142  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9723  | 0                | TYPE_LIMIT | TIF_GTC | 132645e8-5098-45bf-b8bb-e684e7e1ceb4  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9616  | 0                | TYPE_LIMIT | TIF_GTC | 7d776548-7370-4d06-8fae-590ec84c9dc2  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9468  | 0                | TYPE_LIMIT | TIF_GTC | de413d84-3546-4a1b-8a52-6273bfc6c755  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | buy  | 10     | 9662  | 0                | TYPE_LIMIT | TIF_GTC | 5ce15d33-e995-45b4-8979-aebbf9e326ee  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10229 | 0                | TYPE_LIMIT | TIF_GTC | c98a8c72-8994-4a7c-83aa-b6285e1bcea0  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10098 | 0                | TYPE_LIMIT | TIF_GTC | a85f3eb6-9036-48f3-aaa7-fe5a7771f496  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10350 | 0                | TYPE_LIMIT | TIF_GTC | e9c03f76-514d-4eb3-b0a6-25517135722e  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10102 | 0                | TYPE_LIMIT | TIF_GTC | 7fd29cbc-68fe-42a7-8629-e84f96864f94  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10260 | 0                | TYPE_LIMIT | TIF_GTC | 31dbef77-d45c-43a0-8879-dbd359eab40b  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10442 | 0                | TYPE_LIMIT | TIF_GTC | 6b960eac-fc55-4cb7-a2cb-5bf4defe555a  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10363 | 0                | TYPE_LIMIT | TIF_GTC | 7985cd43-3795-4164-aa12-15d90bc027ef  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10408 | 0                | TYPE_LIMIT | TIF_GTC | 8aeb6878-67cb-491f-84c7-640c80f585c4  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10297 | 0                | TYPE_LIMIT | TIF_GTC | 2c6d3691-2dc7-421c-8b8a-94c36342ac62  |            |
      | fa6d991764c5bbba6b895f75d67968e61599ea8c013712c7e57d84c0680dff5f | BTC/ETH   | sell | 10     | 10437 | 0                | TYPE_LIMIT | TIF_GTC | 3997729b-153d-4a45-9d5a-a86aab9c3768  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9568  | 0                | TYPE_LIMIT | TIF_GTC | b6a827f3-363a-4c56-9396-09e8eeac8aec  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9873  | 0                | TYPE_LIMIT | TIF_GTC | 1d77acb4-2939-4ebb-b50a-dd45fbe96caf  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9771  | 0                | TYPE_LIMIT | TIF_GTC | e96f5a0f-144c-4abc-815a-ad979069e31c  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9684  | 0                | TYPE_LIMIT | TIF_GTC | 8f0a89fe-4148-47fb-a7b7-6790dff4c3b4  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9640  | 0                | TYPE_LIMIT | TIF_GTC | d866eb48-7428-49cd-8fcc-00d97b41ea79  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9760  | 0                | TYPE_LIMIT | TIF_GTC | 13b10388-62ce-454c-bed5-a71924e6b71a  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9557  | 0                | TYPE_LIMIT | TIF_GTC | 1c262488-d74f-4053-ad2a-8e2f22d9299b  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9569  | 0                | TYPE_LIMIT | TIF_GTC | dc9c4dfc-6332-4869-9812-e40b7f5d3acc  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9738  | 0                | TYPE_LIMIT | TIF_GTC | 4031164a-9a9b-4e9e-a5b0-c434bb889cba  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | buy  | 10     | 9481  | 0                | TYPE_LIMIT | TIF_GTC | 80c0cdcf-1076-4bf9-b12f-20a0c47473f1  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10193 | 0                | TYPE_LIMIT | TIF_GTC | 2bb605bb-9040-4a83-99ff-5c251e5d4a4a  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10231 | 0                | TYPE_LIMIT | TIF_GTC | d85faa38-96fc-4cbd-b8e0-dc1a77b8d33d  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 9858  | 0                | TYPE_LIMIT | TIF_GTC | d99eb4a1-54bb-4637-9363-017ef853813b  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10291 | 0                | TYPE_LIMIT | TIF_GTC | 205f8d0d-820e-4ac2-b614-f0b9f72b41a4  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10169 | 0                | TYPE_LIMIT | TIF_GTC | 4596154c-f79f-4d8e-81f6-f78a0d2401d9  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10251 | 0                | TYPE_LIMIT | TIF_GTC | 2574873e-59ac-43fa-adbc-365290d37380  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10233 | 0                | TYPE_LIMIT | TIF_GTC | 2bee7727-a865-43e4-ad93-caadc7d90085  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10235 | 0                | TYPE_LIMIT | TIF_GTC | 689a6fab-f98a-49c7-a095-56ff87f57cfc  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10210 | 0                | TYPE_LIMIT | TIF_GTC | f9ca0edc-7458-4d5b-add7-a50632b4f120  |            |
      | f5423f24affe61a6969dba5d54fe8d23590f0d625903fd56bc2a59c7bf198477 | BTC/ETH   | sell | 10     | 10292 | 0                | TYPE_LIMIT | TIF_GTC | 976fa120-69e5-4fb7-b3db-4e439876becf  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9624  | 0                | TYPE_LIMIT | TIF_GTC | 3de46c3d-e124-40ab-92d5-518fa57fafec  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9372  | 0                | TYPE_LIMIT | TIF_GTC | a3e9919e-3055-4397-8faa-fcba25d0a1cb  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9758  | 0                | TYPE_LIMIT | TIF_GTC | 474cb156-81db-4dc8-93e7-5eb2db796a46  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9448  | 0                | TYPE_LIMIT | TIF_GTC | d9918200-a7d1-4771-abd3-f8be5ad72407  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9624  | 0                | TYPE_LIMIT | TIF_GTC | b1d7728e-9007-4958-a9c8-6c0104b9825f  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9842  | 0                | TYPE_LIMIT | TIF_GTC | 9e339fe0-ece7-48e8-b046-134dfdd49d4f  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9464  | 0                | TYPE_LIMIT | TIF_GTC | 04ce51e6-7602-4e74-b45f-7413405ac94a  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9622  | 0                | TYPE_LIMIT | TIF_GTC | 985b1399-afd3-436c-810b-46605471e744  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9717  | 0                | TYPE_LIMIT | TIF_GTC | 6e63b3bd-c8df-4cd6-9f77-6539f247661c  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | buy  | 10     | 9855  | 0                | TYPE_LIMIT | TIF_GTC | 3cbb0988-f94d-47ee-8b4a-89a118b33a44  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10415 | 0                | TYPE_LIMIT | TIF_GTC | bcab8c88-d7d1-4d6c-b30a-448183f5bcb2  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10397 | 0                | TYPE_LIMIT | TIF_GTC | bac912bb-f23c-4a3f-b1d1-3cf50ceddb89  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10342 | 0                | TYPE_LIMIT | TIF_GTC | d3e5335b-cd5c-47c3-9871-ca69aa92dfe4  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10436 | 0                | TYPE_LIMIT | TIF_GTC | 7f5be32e-8dc9-4e23-b2e2-6bda22577287  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10372 | 0                | TYPE_LIMIT | TIF_GTC | 33eea1a2-3edd-4574-8bb1-879ba0f1e460  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10400 | 0                | TYPE_LIMIT | TIF_GTC | fe0b80dd-11ce-4d9d-828a-348e255af7af  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10283 | 0                | TYPE_LIMIT | TIF_GTC | 0895237a-2492-4d29-965c-0ce05c3932ad  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10329 | 0                | TYPE_LIMIT | TIF_GTC | 055af279-dc07-45d7-9e07-940196f59758  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10291 | 0                | TYPE_LIMIT | TIF_GTC | c910ddc0-e889-4c7a-b94a-690a34078c9e  |            |
      | 7954267bca96f9b536c2810a18ffb7bb892a6ca8d62a7ae09a727fe57227d504 | BTC/ETH   | sell | 10     | 10400 | 0                | TYPE_LIMIT | TIF_GTC | 47154119-7706-4d69-98fa-125f5ab9034a  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9735  | 0                | TYPE_LIMIT | TIF_GTC | b8e212ef-efca-4d6e-a8fb-93ba6c2379c4  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9837  | 0                | TYPE_LIMIT | TIF_GTC | a3dcc1ed-3ad6-458f-acdb-00552057ffd6  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9784  | 0                | TYPE_LIMIT | TIF_GTC | f0df78f6-653f-44d9-b368-c6e90b8f3efe  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9600  | 0                | TYPE_LIMIT | TIF_GTC | 5274982d-e10a-47ee-b47b-44c6a4110a46  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9709  | 0                | TYPE_LIMIT | TIF_GTC | 255b5f0f-9cdf-4618-ab1a-6d3457eff309  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9330  | 0                | TYPE_LIMIT | TIF_GTC | 2774e2ff-f53e-4723-bf9b-cf6b8a538a62  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9754  | 0                | TYPE_LIMIT | TIF_GTC | ff7c94fc-d520-4171-a51d-455d40145d6f  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9270  | 0                | TYPE_LIMIT | TIF_GTC | 679cc8df-bdf9-4140-93c1-e9d1dd5b5947  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9393  | 0                | TYPE_LIMIT | TIF_GTC | 1e1e9802-723f-48ca-bf76-cec83fdee466  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | buy  | 10     | 9715  | 0                | TYPE_LIMIT | TIF_GTC | b66aab5b-3020-4d16-916d-2994a503634d  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10315 | 0                | TYPE_LIMIT | TIF_GTC | a83ecb0f-c409-4067-ab7e-5e99a7e37de1  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10553 | 0                | TYPE_LIMIT | TIF_GTC | 081dddb8-c888-45e1-a1d5-9912d0c944fb  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10180 | 0                | TYPE_LIMIT | TIF_GTC | 788cb44a-8d23-4227-af0e-03474d34c2f5  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10329 | 0                | TYPE_LIMIT | TIF_GTC | 834476b2-60ec-4d85-b29f-5cde0319a4a2  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10394 | 0                | TYPE_LIMIT | TIF_GTC | ac04e8f5-d9ef-4e7d-bc10-f694882f814f  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10764 | 0                | TYPE_LIMIT | TIF_GTC | b14fe231-8c44-45ab-9e41-e0728be23ce4  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 9982  | 0                | TYPE_LIMIT | TIF_GTC | 71e05ea4-2c06-48e0-9bcf-b737c56b1528  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10393 | 0                | TYPE_LIMIT | TIF_GTC | 0d1b9f7f-5dfa-411f-a6c7-44d48b0a9a4a  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10400 | 0                | TYPE_LIMIT | TIF_GTC | 1feb243d-bba6-4a04-b673-af084023b931  |            |
      | 6d67b3f68ffa8508fbcc3d4450f00f8464f527f968f5db753f1ff702201b3715 | BTC/ETH   | sell | 10     | 10259 | 0                | TYPE_LIMIT | TIF_GTC | 65089b85-94d7-4411-aeb3-39d2bdf9466d  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9606  | 0                | TYPE_LIMIT | TIF_GTC | d9f14d3d-d4ac-4bed-a6bb-296bc27eb617  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9508  | 0                | TYPE_LIMIT | TIF_GTC | 300d6386-5464-43dd-bbf3-9f6a44a5b983  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9816  | 0                | TYPE_LIMIT | TIF_GTC | 925c99d0-8219-4a4d-b333-7481c779821b  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9320  | 0                | TYPE_LIMIT | TIF_GTC | 46c14e21-112b-4310-9d87-58eb169ccabc  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9288  | 0                | TYPE_LIMIT | TIF_GTC | 46b08f64-9211-4438-8693-9ee8c450a624  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9751  | 0                | TYPE_LIMIT | TIF_GTC | 88b22cd9-3505-46ba-81de-c0ef011bd0ee  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9681  | 0                | TYPE_LIMIT | TIF_GTC | 49b20eb6-79fe-4ca2-a1a6-0fa8b8ced406  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9805  | 0                | TYPE_LIMIT | TIF_GTC | 22a58089-27e2-4452-b6d7-9d649b0e31e3  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9791  | 0                | TYPE_LIMIT | TIF_GTC | 95c84328-f9ff-4f30-8fa6-f066d1b1af8d  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | buy  | 10     | 9608  | 0                | TYPE_LIMIT | TIF_GTC | 91189e0a-ee68-415d-805d-3a2f879dfe5a  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10140 | 0                | TYPE_LIMIT | TIF_GTC | 5df5f522-65e8-46d6-8fba-1161202412e4  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10384 | 0                | TYPE_LIMIT | TIF_GTC | 7fed2d56-223d-482f-ac83-bb0c896d8f9c  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10067 | 0                | TYPE_LIMIT | TIF_GTC | 15982ede-a5a0-42e9-b324-8ce2e45a65ac  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10357 | 0                | TYPE_LIMIT | TIF_GTC | 137155c0-76de-415b-910a-419c0dec7a87  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10133 | 0                | TYPE_LIMIT | TIF_GTC | 3c7d2289-5a90-4c38-bb57-aabf1d79adab  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10363 | 0                | TYPE_LIMIT | TIF_GTC | 57288912-e0fa-42dc-951b-b1d4524f76dd  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10365 | 0                | TYPE_LIMIT | TIF_GTC | 7cba4449-3701-4fc6-8790-207de7ea3a9c  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 9968  | 0                | TYPE_LIMIT | TIF_GTC | ad499e22-21cb-42eb-97ed-9b0f4c3041dd  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10389 | 0                | TYPE_LIMIT | TIF_GTC | 88833e27-a6cc-4c94-811b-0cc44fbe8be7  |            |
      | 8d375cd3718d38a2c9de64c157833aadaf94c5b32a692778724056a92373d7a6 | BTC/ETH   | sell | 10     | 10251 | 0                | TYPE_LIMIT | TIF_GTC | 20f0a29f-4149-4d06-95ae-f29e85368b76  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9820  | 0                | TYPE_LIMIT | TIF_GTC | 56d5680b-4c77-492d-8642-56bfc3c93539  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9810  | 0                | TYPE_LIMIT | TIF_GTC | e0935051-f270-4c60-bebf-e96c34b787ab  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9653  | 0                | TYPE_LIMIT | TIF_GTC | 37d0134d-599f-4138-8090-5798d3ec48e1  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9600  | 0                | TYPE_LIMIT | TIF_GTC | 2b56be13-a68e-43db-a8a7-3d8f210b9b2c  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9655  | 0                | TYPE_LIMIT | TIF_GTC | 66a1dbd7-9816-43a0-9318-a050c918d420  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9855  | 0                | TYPE_LIMIT | TIF_GTC | 3a5682fc-ec54-41a6-a450-73275cd76c10  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9616  | 0                | TYPE_LIMIT | TIF_GTC | c9fcf0b5-a937-4828-8496-62fcae2a010a  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9859  | 0                | TYPE_LIMIT | TIF_GTC | 81006c10-8661-4f85-90b0-a9a12fa746f6  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9663  | 0                | TYPE_LIMIT | TIF_GTC | c7e19e88-afe4-4065-9997-82ed9ef995d5  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | buy  | 10     | 9180  | 0                | TYPE_LIMIT | TIF_GTC | 92082e44-ccc4-4685-9264-915d43c99179  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10041 | 0                | TYPE_LIMIT | TIF_GTC | ccfe3bf8-15bd-46bb-b9a8-7832bf46c782  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10185 | 0                | TYPE_LIMIT | TIF_GTC | 7cfe533a-93fb-4e0b-828c-559077141f2e  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10294 | 0                | TYPE_LIMIT | TIF_GTC | ed362e1a-8eb4-4404-8cea-9be845c8afb0  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10180 | 0                | TYPE_LIMIT | TIF_GTC | 2db1b73f-2962-495a-b8c9-63e00ad8e707  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10565 | 0                | TYPE_LIMIT | TIF_GTC | 8d0e2b29-f55f-497a-803a-4e18cf8ae253  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10201 | 0                | TYPE_LIMIT | TIF_GTC | 32c26773-f548-4523-86e3-fc5cd9518000  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10287 | 0                | TYPE_LIMIT | TIF_GTC | 37453702-1518-4a98-84fb-3919bbd75dcc  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10280 | 0                | TYPE_LIMIT | TIF_GTC | e90cf98d-85b3-4be4-a3df-9a2c954679aa  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10472 | 0                | TYPE_LIMIT | TIF_GTC | 4db58179-8acb-4826-92e7-084785bcfa61  |            |
      | ab7ef99c70faa3e3bdc3ac9e4c7b994722f4bce963172f49cfbacf7cbdd253b2 | BTC/ETH   | sell | 10     | 10296 | 0                | TYPE_LIMIT | TIF_GTC | abb687b9-4659-45e2-93f4-a3a170fe9d2e  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9724  | 0                | TYPE_LIMIT | TIF_GTC | 6817a034-61c8-4cfc-8430-ece47823efd1  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 10036 | 0                | TYPE_LIMIT | TIF_GTC | 5f4b12d9-0b90-4303-af88-82f83744bb4d  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9854  | 0                | TYPE_LIMIT | TIF_GTC | 9608571f-b6b4-4d8c-a01b-aea79f495fde  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9838  | 0                | TYPE_LIMIT | TIF_GTC | dca6a0a8-413d-4284-aa05-1d95f4764da6  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9901  | 0                | TYPE_LIMIT | TIF_GTC | 4e4b3ac9-2cdd-47bd-9a14-96fb97405bfb  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9495  | 0                | TYPE_LIMIT | TIF_GTC | b24e418e-bb8a-4c5c-be56-595b5989530b  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9434  | 0                | TYPE_LIMIT | TIF_GTC | 0c845990-9681-48e6-a2d3-a523d86e90f8  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9775  | 0                | TYPE_LIMIT | TIF_GTC | d055d951-7f02-46e2-9049-289fb19e13e9  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9767  | 0                | TYPE_LIMIT | TIF_GTC | 4013dfb1-edbc-48b0-b74c-f51bf15a1ed6  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | buy  | 10     | 9857  | 0                | TYPE_LIMIT | TIF_GTC | 51a2c6aa-22e4-4681-8d32-40328a34a073  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10317 | 0                | TYPE_LIMIT | TIF_GTC | 58c6afc0-4abc-49de-907c-2d00de718cfc  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10308 | 0                | TYPE_LIMIT | TIF_GTC | 03451858-a44f-46d8-b285-1f2e6d9db727  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10284 | 0                | TYPE_LIMIT | TIF_GTC | d01401f1-6279-4c80-89e8-6616ab60bc57  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10446 | 0                | TYPE_LIMIT | TIF_GTC | 3518bc13-5b00-4f92-9c43-47cb0d3fcad1  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10812 | 0                | TYPE_LIMIT | TIF_GTC | f4db6a55-ffef-41a0-86fd-afb2a75519ce  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10442 | 0                | TYPE_LIMIT | TIF_GTC | b3cf93c5-74ef-4965-8707-51986ba7e71f  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10349 | 0                | TYPE_LIMIT | TIF_GTC | 162d30c7-28b1-464f-beb9-f442241f4f02  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10434 | 0                | TYPE_LIMIT | TIF_GTC | 673e5970-58fb-4615-b842-19dc2e5ece13  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10361 | 0                | TYPE_LIMIT | TIF_GTC | 5bc6a9b4-355a-4317-a0f0-0be3067cdd23  |            |
      | 92f02bcabba011f248fd06c9e1f4e94066d7c3c23afed21a0540c2ee07409209 | BTC/ETH   | sell | 10     | 10334 | 0                | TYPE_LIMIT | TIF_GTC | 144efd4a-f2ce-46e6-9326-9289f6a0c0a3  |            |
      | 92bf2b120913090e5f09fa92e96f972a4d325c84fa662ed52ae1aba81e1ba785 | BTC/ETH   | buy  | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTT | f39669f9-67ba-41c5-b576-16998b258fd9  | 120        |
      | fac43f8b56111cac2e135b09122af24fc38dac9321bbc884c9e0f945c351357f | BTC/ETH   | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTT | 7e5e000d-f187-45a5-ad3c-8751c4c116fb  | 120        |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9985  | 0                | TYPE_LIMIT | TIF_GTC | 602c5d0f-961c-410c-967c-8c63d3e39479  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9964  | 0                | TYPE_LIMIT | TIF_GTC | 4c3941a5-7878-479d-b58d-29f54a7d4ef4  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9773  | 0                | TYPE_LIMIT | TIF_GTC | 79d8439f-c75f-41f0-a7f8-15d7ff5db380  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9604  | 0                | TYPE_LIMIT | TIF_GTC | add5fca3-6b8b-4b91-84dc-bcc7aa36c1e6  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9515  | 0                | TYPE_LIMIT | TIF_GTC | 8d1ba765-11b7-4099-a681-11597ae31b8b  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9410  | 0                | TYPE_LIMIT | TIF_GTC | 91686ccb-5ddf-4b9e-bb9a-011bcd266f6f  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 10081 | 0                | TYPE_LIMIT | TIF_GTC | 1dbc96a6-25e2-4e8f-a0a9-3bb42b84f55a  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9444  | 0                | TYPE_LIMIT | TIF_GTC | ff14c508-1786-4ab1-98ef-30cd4a104936  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9669  | 0                | TYPE_LIMIT | TIF_GTC | 89094228-184f-415d-abce-595e5b0614b6  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | buy  | 10     | 9747  | 0                | TYPE_LIMIT | TIF_GTC | a334fb02-e165-4827-9f86-dca8a30bdde0  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10143 | 0                | TYPE_LIMIT | TIF_GTC | 5ff5a378-29f5-46bb-9ffa-54fd94694239  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10241 | 0                | TYPE_LIMIT | TIF_GTC | a5810b6c-4b31-4427-a224-045256dca728  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10330 | 0                | TYPE_LIMIT | TIF_GTC | 34fb75dd-1736-46ed-b046-47faf9b4130a  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10210 | 0                | TYPE_LIMIT | TIF_GTC | 609e83cc-dcaf-418d-86c7-14ff8dd4c86a  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 9972  | 0                | TYPE_LIMIT | TIF_GTC | f84f1b24-7714-4502-94c7-c12e5269d54a  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10171 | 0                | TYPE_LIMIT | TIF_GTC | de628cbf-c08f-46e6-a11c-44274d5dc57c  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10363 | 0                | TYPE_LIMIT | TIF_GTC | 18ce7ae4-104f-4394-acad-2f7abd82884b  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10354 | 0                | TYPE_LIMIT | TIF_GTC | 7909462d-f4a9-423a-b771-79bec6148cd2  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 10267 | 0                | TYPE_LIMIT | TIF_GTC | 25126b13-1492-4554-b2b6-a79f6a1c0114  |            |
      | 94b2a356e3248f0cf344ab247489da0a2f4c522a7a536fb819c49b232484f4e5 | BTC/ETH   | sell | 10     | 9762  | 0                | TYPE_LIMIT | TIF_GTC | c99cb4e9-8217-4ff1-9f22-edcd56687bb4  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9903  | 0                | TYPE_LIMIT | TIF_GTC | 84762945-8ca6-44f4-bc4b-9a1bd03ec26f  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9739  | 0                | TYPE_LIMIT | TIF_GTC | 6da35d6e-8eaf-4716-92e1-b32a186c095f  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9943  | 0                | TYPE_LIMIT | TIF_GTC | e764a66c-65d1-4125-ba06-af1fb03e4373  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9988  | 0                | TYPE_LIMIT | TIF_GTC | 16ffce73-7659-4440-833c-b404a94e3271  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9938  | 0                | TYPE_LIMIT | TIF_GTC | e8b01e41-2c4b-415d-8edb-7a08bcf7b38b  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9593  | 0                | TYPE_LIMIT | TIF_GTC | 0a4e892b-9977-482f-92c7-9ae98c5034f6  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9878  | 0                | TYPE_LIMIT | TIF_GTC | 50203387-041e-4f55-b943-bd24e76e699c  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9554  | 0                | TYPE_LIMIT | TIF_GTC | 2bdf492a-b8a3-4aeb-b493-33152d49cc25  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9762  | 0                | TYPE_LIMIT | TIF_GTC | b03a70b8-195a-44b6-ba74-5adaaaadbfa1  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | buy  | 10     | 9653  | 0                | TYPE_LIMIT | TIF_GTC | 5d703ac7-2602-41c9-964d-5d9b368f338d  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10391 | 0                | TYPE_LIMIT | TIF_GTC | cf159344-7afc-4742-b5c2-697df7fa59cc  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10294 | 0                | TYPE_LIMIT | TIF_GTC | 1b261abd-92a7-4aeb-883f-a5591a91d2d8  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10083 | 0                | TYPE_LIMIT | TIF_GTC | 4c599948-8eb4-4ade-b64b-35b8f0441e08  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10642 | 0                | TYPE_LIMIT | TIF_GTC | 45c5e461-eed3-4453-86b3-f142f8f7d284  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10457 | 0                | TYPE_LIMIT | TIF_GTC | f62cf0b8-eb2c-48f5-9956-5a46182815bb  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10399 | 0                | TYPE_LIMIT | TIF_GTC | b89dce68-9742-4b6f-8f02-ae82629c5e1c  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10564 | 0                | TYPE_LIMIT | TIF_GTC | 7903fc74-ab7d-4e65-8ef0-3c71a25069e8  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10475 | 0                | TYPE_LIMIT | TIF_GTC | 3fbeb4d6-92e7-44c6-8787-e54dce92fe48  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10335 | 0                | TYPE_LIMIT | TIF_GTC | 484c7c03-3f81-4714-9aba-2e724d36a775  |            |
      | 75746fa748071d20a5f1c87397276b0222c2db2a85ef7e667beb1acfbe3b2cf6 | BTC/ETH   | sell | 10     | 10119 | 0                | TYPE_LIMIT | TIF_GTC | b074683e-0955-4e82-b6a3-fcea99aad5d4  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9791  | 0                | TYPE_LIMIT | TIF_GTC | 41686474-5958-4162-9c39-167788b2c7ae  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9582  | 0                | TYPE_LIMIT | TIF_GTC | 25b44c56-1df8-4dd6-8353-893e631ffa33  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9703  | 0                | TYPE_LIMIT | TIF_GTC | af1e448a-76f2-44be-8d6b-62ccfda6f190  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9705  | 0                | TYPE_LIMIT | TIF_GTC | cfe162cc-522e-484f-b315-f327f5bbfcf7  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9476  | 0                | TYPE_LIMIT | TIF_GTC | e378fa18-7ca7-4513-b9b9-f7992aa80e07  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9663  | 0                | TYPE_LIMIT | TIF_GTC | f8aef43b-9201-46e8-82d1-0bca98e09a9d  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9796  | 0                | TYPE_LIMIT | TIF_GTC | fdd97c60-eb2a-4914-9475-8a44c1fff6d8  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9560  | 0                | TYPE_LIMIT | TIF_GTC | 73fb8433-552d-4f6c-862d-73571f8721fa  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9616  | 0                | TYPE_LIMIT | TIF_GTC | e4f0e100-fd90-48e9-bf95-e40670236ce5  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | buy  | 10     | 9754  | 0                | TYPE_LIMIT | TIF_GTC | 5200b283-3787-4426-8f26-aee9d794fa7c  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10122 | 0                | TYPE_LIMIT | TIF_GTC | aeb97fc0-dcf6-4888-b090-971358490f40  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10339 | 0                | TYPE_LIMIT | TIF_GTC | 1b535edb-a4ed-4479-a6c6-8b229a67652d  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10137 | 0                | TYPE_LIMIT | TIF_GTC | 4d0d4ada-6e26-4c74-a6d3-901668d02b4c  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10385 | 0                | TYPE_LIMIT | TIF_GTC | a70435eb-5555-4cd0-a680-e2424c50b84b  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10288 | 0                | TYPE_LIMIT | TIF_GTC | 4569769d-767c-416a-ad15-aedad83f6593  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10045 | 0                | TYPE_LIMIT | TIF_GTC | 42f74e4b-27a0-4171-8fbe-504a979493fa  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 9726  | 0                | TYPE_LIMIT | TIF_GTC | 80b7faf4-681e-4880-961f-c0172d1502fb  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10603 | 0                | TYPE_LIMIT | TIF_GTC | c87376d9-6a7f-414f-8054-1250019ae8ce  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10344 | 0                | TYPE_LIMIT | TIF_GTC | 7691d6de-8c88-4617-96af-f7c9aa22795f  |            |
      | 23f06c0abf339ed372f72be3fd7243a243abd931d90625b42afef63a077c9abf | BTC/ETH   | sell | 10     | 10302 | 0                | TYPE_LIMIT | TIF_GTC | bfd9323b-b31a-4dc5-9325-27b22fef57ac  |            |

    
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "10000" for the market "BTC/ETH"
