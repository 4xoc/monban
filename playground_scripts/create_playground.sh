#!/bin/bash
DIR=`dirname "$0"`

docker run -p 389:389 -p 636:636 --name my-openldap-container --detach osixia/openldap:1.3.0

until docker exec my-openldap-container ldapsearch -x -H ldap://localhost -b cn=config -D "cn=admin,cn=config" -w config > /dev/null 2>&1
do 
    echo -n "."
done

docker cp $DIR/people_groups.ldif my-openldap-container:/tmp/
docker cp $DIR/sudo.schema.ldif my-openldap-container:/tmp/
docker exec my-openldap-container ldapmodify -x -H ldap://localhost -w config -D "cn=admin,cn=config" -a -f /tmp/sudo.schema.ldif
docker exec my-openldap-container ldapmodify -x -H ldap://localhost -w admin -D "cn=admin,dc=example,dc=org" -a -f /tmp/people_groups.ldif
echo "OpenLDAP up and running. You can now try out monban. Enjoy responsibly..."
