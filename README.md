# godubbo
golang dubbo , hessian over tcp


Step 1
prepare local zookeeper server: 127.0.0.1:2181

Step 2
java -jar godubbo-server.jar

Step 3 
go run example.go



java dubbo server interface:


    package zh.springboot.service;

    public interface BasicService {

	  public void test(String a);

	  public String test(Integer a);

	  public String test(Integer a, String b);

	  public void test2();
	
	  //todo, not support now
	  public String[] demo(String[] s) ;
    }

 
    public interface ClassService {
    
	  String sayHello(String name);

	  String helloWorld();

	  Address getAddressDefalut();

	  Address getAddress(Address address);

      Person getPersonDefalut();

      Person getPerson(String name, Address address, int age);

      Person getPerson2(Person p);

	  List<Address> getAddresses();

	  List<Person> getPersons();

	  Map<String, Person> getPersonMap();

      List<Person> getPersons2(List<Person> persons);

    }

java class object:

    package zh.springboot.service;

    public class Person implements Serializable {

  	private int age;
   
  	private String name;
  
  	private Address address;
  
     }
  
    public class Address implements Serializable {

   	private String city;
  
  	private String country;
  
    }
  
  
