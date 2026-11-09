<?php
    # Product name
    $config['product_name'] = 'BillionMail';

    $config['mime_types'] = '/var/roundcube/config/mime.types';
       
    # Plugins
    $config['plugins'] = array('password','userinfo','newmail_notifier','emoticons','zipdownload','managesieve');

    # Dovecot ManageSieve: server-side mailbox filters (internal Docker network).
    # STARTTLS is negotiated over the private BillionMail bridge network.
    $config['managesieve_host'] = 'tls://dovecot:4190';

    # Password
    $config['password_query'] = 'update mailbox set password = %P where username = %u';
    $config['password_algorithm'] = 'md5-crypt';


# Please create an extra file: extra.php. Used to persist overrides Configuration
